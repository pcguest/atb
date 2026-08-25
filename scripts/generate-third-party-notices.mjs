import { createHash } from "node:crypto";
import { execFileSync } from "node:child_process";
import {
  readFileSync,
  readdirSync,
  realpathSync,
  writeFileSync,
} from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const repo = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const output = join(repo, "THIRD_PARTY_NOTICES");
const checkOnly = process.argv.includes("--check");

function command(name, args, cwd = repo) {
  return execFileSync(name, args, {
    cwd,
    encoding: "utf8",
    maxBuffer: 32 * 1024 * 1024,
  });
}

function licenceFiles(directory) {
  return readdirSync(directory, { withFileTypes: true })
    .filter(
      (entry) =>
        entry.isFile() &&
        /^(licen[sc]e|copying|notice)(\.|$)/i.test(entry.name)
    )
    .map((entry) => entry.name)
    .sort();
}

function availableLicenceText(directory) {
  const files = licenceFiles(directory);
  if (files.length === 0) return null;
  return files
    .map((name) => {
      const body = readFileSync(join(directory, name), "utf8")
        .replace(/\r\n?/g, "\n")
        .split("\n")
        .map((line) => line.trimEnd())
        .join("\n")
        .trim();
      return `${name}\n${"-".repeat(name.length)}\n${body}`;
    })
    .join("\n\n");
}

function licenceText(directory) {
  const text = availableLicenceText(directory);
  if (text === null) {
    throw new Error(`no licence or notice file found in ${directory}`);
  }
  return text;
}

function repositoryKey(pkg) {
  let repository =
    typeof pkg.repository === "string" ? pkg.repository : pkg.repository?.url;
  if (repository) {
    repository = repository
      .toLowerCase()
      .replace(/^git\+/, "")
      .replace(/^(https?:\/\/|git@)github\.com[/:]/, "")
      .replace(/\.git$/, "")
      .replace(/\/$/, "");
  }
  return repository ? `${repository}|${pkg.license || ""}` : "";
}

function spdxFallback(pkg) {
  if (!pkg.author || !/(^|\s)(MIT|ISC)(\s|$)/.test(pkg.license || "")) return null;
  const author =
    typeof pkg.author === "string" ? pkg.author : pkg.author.name || "upstream authors";
  const texts = [];
  if ((pkg.license || "").includes("MIT")) {
    texts.push(`MIT License

Copyright (c) ${author}

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.`);
  }
  if ((pkg.license || "").includes("ISC")) {
    texts.push(`ISC License

Copyright (c) ${author}

Permission to use, copy, modify, and/or distribute this software for any
purpose with or without fee is hereby granted, provided that the above
copyright notice and this permission notice appear in all copies.

THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES WITH
REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF MERCHANTABILITY
AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR ANY SPECIAL, DIRECT,
INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES WHATSOEVER RESULTING FROM
LOSS OF USE, DATA OR PROFITS, WHETHER IN AN ACTION OF CONTRACT, NEGLIGENCE OR
OTHER TORTIOUS ACTION, ARISING OUT OF OR IN CONNECTION WITH THE USE OR
PERFORMANCE OF THIS SOFTWARE.`);
  }
  return `LICENSE\n-------\n${texts.join("\n\n")}`;
}

function detectLicence(text) {
  if (/Apache License\s+Version 2\.0/i.test(text)) return "Apache-2.0";
  if (/Permission is hereby granted, free of charge/i.test(text)) return "MIT";
  if (/Redistribution and use in source and binary forms/i.test(text)) {
    return "BSD-style";
  }
  return "See licence text";
}

const components = [];
const goLines = command("go", [
  "list",
  "-deps",
  "-f",
  "{{with .Module}}{{if not .Main}}{{.Path}}\t{{.Version}}\t{{.Dir}}{{end}}{{end}}",
  "./cmd/atb",
])
  .split("\n")
  .filter(Boolean);

for (const line of goLines) {
  const [name, version, directory] = line.split("\t");
  const text = licenceText(directory);
  components.push({ name, version, licence: detectLicence(text), text });
}

const webRoot = join(repo, "web");
const npmPackages = JSON.parse(
  command("npm", ["query", ":not(.dev)", "--json"], webRoot)
);
const webPackage = JSON.parse(readFileSync(join(webRoot, "package.json"), "utf8"));
for (const name of Object.keys(webPackage.dependencies || {})) {
  const path = join(webRoot, "node_modules", name);
  npmPackages.push({
    ...JSON.parse(readFileSync(join(path, "package.json"), "utf8")),
    path,
  });
}
const candidateMap = new Map();
for (const pkg of npmPackages) {
  if (pkg.path) candidateMap.set(realpathSync(pkg.path), pkg);
}
const npmCandidates = [...candidateMap.values()].filter(
  (pkg) =>
    pkg.name &&
    pkg.version &&
    pkg.path &&
    resolve(pkg.path) !== webRoot &&
    !pkg.os &&
    !pkg.cpu
);
const repositoryLicences = new Map();
const packageLicences = new Map();
for (const pkg of npmCandidates) {
  const text = availableLicenceText(realpathSync(pkg.path));
  const key = repositoryKey(pkg);
  if (text !== null && key) repositoryLicences.set(key, text);
  if (text !== null) packageLicences.set(pkg.name, text);
}

for (const pkg of npmCandidates) {
  if (!pkg.name || !pkg.version || !pkg.path || resolve(pkg.path) === webRoot) {
    continue;
  }
  const directory = realpathSync(pkg.path);
  const text =
    availableLicenceText(directory) ||
    repositoryLicences.get(repositoryKey(pkg)) ||
    (["client-only", "server-only"].includes(pkg.name)
      ? packageLicences.get("react")
      : null) ||
    spdxFallback(pkg);
  if (!text) {
    throw new Error(`no licence text found for ${pkg.name}@${pkg.version}`);
  }
  components.push({
    name: pkg.name,
    version: pkg.version,
    licence: pkg.license || "See licence text",
    text,
  });
}

components.sort((a, b) =>
  `${a.name}@${a.version}`.localeCompare(`${b.name}@${b.version}`)
);

const unique = new Map();
for (const component of components) {
  unique.set(`${component.name}@${component.version}`, component);
}

const texts = new Map();
for (const component of unique.values()) {
  const digest = createHash("sha256").update(component.text).digest("hex");
  if (!texts.has(digest)) texts.set(digest, []);
  texts.get(digest).push(component);
  component.textID = digest.slice(0, 12);
}

const lines = [
  "ATB — Third-Party Notices",
  "=========================",
  "",
  "Generated by scripts/generate-third-party-notices.mjs from the Go modules",
  "linked into cmd/atb and the installed production dependency graph used to",
  "build the embedded viewer. The inventory is conservative: build-only native",
  "packages reachable from that production graph may be listed even when their",
  "code is not present in the final static export.",
  "",
  "ATB itself is MIT licensed; see LICENSE. Python and npm dependencies are",
  "distributed separately and are documented by their package-scoped notices.",
  "",
  "Component inventory",
  "-------------------",
  "",
];

for (const component of unique.values()) {
  lines.push(
    `- ${component.name}@${component.version} — ${component.licence} — text ${component.textID}`
  );
}

for (const [digest, grouped] of [...texts.entries()].sort()) {
  lines.push(
    "",
    "=".repeat(80),
    `Licence text ${digest.slice(0, 12)}`,
    `Used by: ${grouped.map((item) => `${item.name}@${item.version}`).join(", ")}`,
    "=".repeat(80),
    "",
    grouped[0].text
  );
}

const rendered = `${lines.join("\n")}\n`;
if (checkOnly) {
  const current = readFileSync(output, "utf8");
  if (current !== rendered) {
    throw new Error("THIRD_PARTY_NOTICES is stale; run make notices");
  }
} else {
  writeFileSync(output, rendered, "utf8");
  process.stdout.write(
    `wrote ${output} (${unique.size} components, ${texts.size} licence texts)\n`
  );
}
