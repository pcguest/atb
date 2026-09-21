#!/usr/bin/env node
/**
 * ATB TypeScript SDK Quickstart
 *
 * This demonstrates a minimal bundle creation workflow that produces
 * a verifiable ATB bundle.
 *
 * Run:
 *   cd examples/typescript
 *   npx -y tsx quickstart.ts
 *
 * Or with global install:
 *   npm install -g tsx @pcguest/atb-sdk
 *   tsx quickstart.ts
 */

import {
  AI_MODEL_INVOKED_EVENT_TYPE,
  AI_REQUEST_RECEIVED_EVENT_TYPE,
  Bundle,
} from "@pcguest/atb-sdk";
import { existsSync, mkdirSync } from "node:fs";
import { resolve } from "node:path";

const BUNDLE_DIR = "run.atb";
const BUNDLE_PATH = resolve(BUNDLE_DIR, "bundle.atb");

async function main() {
  console.log("Creating ATB bundle...");

  // Ensure output directory exists
  if (!existsSync(BUNDLE_DIR)) {
    mkdirSync(BUNDLE_DIR, { recursive: true });
  }

  const bundle = new Bundle();

  // 1. Record incoming request
  bundle.append(AI_REQUEST_RECEIVED_EVENT_TYPE, {
    request_id: "req-001",
    actor_id_hash: "sha256:actor-001",
    purpose_tag: "quickstart_demo",
  });
  console.log("✓ Appended event #1 [ai.request.received]");

  // 2. Record model invocation
  bundle.append(AI_MODEL_INVOKED_EVENT_TYPE, {
    model_provider: "openai",
    model_id: "gpt-4o-mini",
    model_parameters_digest: "sha256:params-abc",
    prompt_digest: "sha256:prompt-def",
  });
  console.log("✓ Appended event #2 [ai.model.invoked]");

  // Save bundle
  await bundle.save(BUNDLE_PATH);
  console.log(`\nBundle saved to: ${BUNDLE_PATH}`);

  // Verify local hash chain
  const loaded = Bundle.load(BUNDLE_PATH);
  const result = loaded.verify();
  console.log(`\nVerification complete`);
  console.log(`Signatures found: ${result.signatures.length}`);
  console.log(`Records in bundle: ${loaded.records.length}`);

  console.log("\nFor full profile evaluation, run:");
  console.log(`  atb verify --bundle ${BUNDLE_PATH} --profile atb.profile.rag_answer`);
}

main().catch((error) => {
  console.error("Error:", error);
  process.exit(1);
});
