#!/usr/bin/env node
// Two-signal Node dependency security gate.
//
//   runtime     release-blocking: HIGH/CRITICAL in production dependencies fail.
//   visibility  full dependency tree: HIGH/CRITICAL advisories are reported,
//               including development/build-only ones, but do not fail the run.
//
// Both modes fail closed on an audit execution failure. `npm audit` exits with
// code 1 both when advisories are found and when the audit itself cannot run,
// so the exit code alone is ambiguous; the JSON `error` field is the reliable
// discriminator between AUDIT_EXECUTION_FAILURE and AUDIT_COMPLETED_WITH_ADVISORIES.
//
// Exit codes:
//   0  audit completed; no blocking condition
//   1  audit completed; runtime HIGH/CRITICAL found (release-blocking)
//   2  audit execution failure (registry/network/tool/lockfile error)
//
// Usage:
//   node scripts/npm-audit-gate.mjs <runtime|visibility> [--report <path>]
//
// --report reads a captured `npm audit --json` document instead of invoking
// npm; it exists solely for deterministic gate tests.

import { readFileSync } from "node:fs";
import { spawnSync } from "node:child_process";

const EXIT_PASS = 0;
const EXIT_RUNTIME_BLOCK = 1;
const EXIT_EXECUTION_FAILURE = 2;

const argv = process.argv.slice(2);
const mode = argv[0];
let reportPath = null;
const reportFlag = argv.indexOf("--report");
if (reportFlag !== -1) {
  reportPath = argv[reportFlag + 1] ?? null;
}

if (mode !== "runtime" && mode !== "visibility") {
  console.error("usage: npm-audit-gate.mjs <runtime|visibility> [--report <path>]");
  process.exit(EXIT_EXECUTION_FAILURE);
}

const omitDev = mode === "runtime";

function loadReport() {
  if (reportPath !== null) {
    try {
      return readFileSync(reportPath, "utf8");
    } catch (err) {
      console.error(`AUDIT_EXECUTION_FAILURE: cannot read report ${reportPath}: ${err.message}`);
      process.exit(EXIT_EXECUTION_FAILURE);
    }
  }

  const args = ["audit", "--json"];
  if (omitDev) args.push("--omit=dev");

  const run = spawnSync("npm", args, { encoding: "utf8" });
  if (run.error) {
    console.error(`AUDIT_EXECUTION_FAILURE: cannot run npm: ${run.error.message}`);
    process.exit(EXIT_EXECUTION_FAILURE);
  }
  if (run.status !== 0) {
    // Non-zero is expected when advisories exist; the JSON body decides why.
    console.error(`npm audit --json exited ${run.status}; inspecting JSON report`);
  }
  if ((run.stderr || "").trim()) console.error(run.stderr.trim());
  return run.stdout || "";
}

let report;
try {
  report = JSON.parse(loadReport());
} catch {
  console.error(`AUDIT_EXECUTION_FAILURE: npm audit did not return parseable JSON (${mode})`);
  process.exit(EXIT_EXECUTION_FAILURE);
}

if (report && report.error) {
  const code = report.error.code || "unknown";
  const summary = report.error.summary || "no summary";
  console.error(`AUDIT_EXECUTION_FAILURE: ${code}: ${summary}`);
  process.exit(EXIT_EXECUTION_FAILURE);
}

const counts = report?.metadata?.vulnerabilities;
if (!counts || typeof counts !== "object") {
  console.error(`AUDIT_EXECUTION_FAILURE: audit report missing metadata.vulnerabilities (${mode})`);
  process.exit(EXIT_EXECUTION_FAILURE);
}

const high = counts.high || 0;
const critical = counts.critical || 0;
const scope = omitDev ? "production/runtime dependencies" : "full dependency tree";

console.log(
  `npm audit (${scope}): ${counts.total || 0} vulnerable ` +
    `(info ${counts.info || 0}, low ${counts.low || 0}, moderate ${counts.moderate || 0}, ` +
    `high ${high}, critical ${critical})`,
);

function highCriticalNames() {
  return Object.entries(report.vulnerabilities || {})
    .filter(([, v]) => v.severity === "high" || v.severity === "critical")
    .map(([name, v]) => {
      const via = (v.via || [])
        .filter((x) => x && typeof x === "object" && x.title)
        .map((x) => x.title)
        .join("; ");
      return via ? `  - ${name} (${v.severity}): ${via}` : `  - ${name} (${v.severity})`;
    })
    .sort();
}

if (mode === "runtime") {
  if (high > 0 || critical > 0) {
    console.error(`RUNTIME_HIGH_CRITICAL: ${high} high, ${critical} critical in production dependencies — release-blocking`);
    for (const line of highCriticalNames()) console.error(line);
    process.exit(EXIT_RUNTIME_BLOCK);
  }
  console.log("Runtime audit clean: no HIGH/CRITICAL in production dependencies.");
  process.exit(EXIT_PASS);
}

// visibility: report, never block on advisories, fail only on execution failure above.
if (high > 0 || critical > 0) {
  console.error(
    `DEVELOPMENT_VISIBILITY: unresolved HIGH/CRITICAL advisories present ` +
      `(${high} high, ${critical} critical). Visible for review; runtime enforcement is a separate signal.`,
  );
  for (const line of highCriticalNames()) console.error(line);
  console.error(
    "Development/build dependencies are not automatically release-blocking, but they can affect " +
      "developer workstations, CI, build systems, and generated artefacts, so they remain tracked.",
  );
} else {
  console.log("Full-tree audit clean: no HIGH/CRITICAL advisories across development and runtime dependencies.");
}

process.exit(EXIT_PASS);
