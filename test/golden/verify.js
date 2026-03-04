#!/usr/bin/env node
"use strict";

const { canonicalize } = require("../../sdk/typescript/dist/index.js");
const { createHash } = require("node:crypto");
const { readFileSync, writeFileSync } = require("node:fs");

const GENESIS_HASH = "0".repeat(64);

const event = JSON.parse(readFileSync("input.json", "utf-8"));
const canonical = canonicalize(event);
const digest = createHash("sha256")
  .update(GENESIS_HASH, "utf-8")
  .update(canonical, "utf-8")
  .digest("hex");

writeFileSync("output-typescript.json", canonical, "utf-8");
writeFileSync("hash-typescript.txt", digest, "utf-8");

const goCanonical = readFileSync("output-go.json", "utf-8");
const goDigest = readFileSync("hash-go.txt", "utf-8");

if (canonical !== goCanonical) {
  console.error("TypeScript canonical output mismatch with Go.");
  console.error(`typescript: ${canonical}`);
  console.error(`go:         ${goCanonical}`);
  process.exit(1);
}

if (digest !== goDigest) {
  console.error("TypeScript hash mismatch with Go.");
  console.error(`typescript: ${digest}`);
  console.error(`go:         ${goDigest}`);
  process.exit(1);
}

console.log("✅ TypeScript matches Go (byte-for-byte)");
console.log(`TypeScript hash: ${digest}`);
