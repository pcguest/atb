#!/usr/bin/env node
"use strict";

const { encryptRaw } = require("../../sdk/typescript/dist/index.js");

function required(name) {
  const value = process.env[name];
  if (!value) {
    throw new Error(`missing required env var ${name}`);
  }
  return value;
}

async function main() {
  const plaintext = Buffer.from(
    required("ATB_PARITY_PLAINTEXT_B64"),
    "base64"
  );
  const password = required("ATB_PARITY_PASSWORD");
  const salt = Buffer.from(required("ATB_PARITY_SALT_HEX"), "hex");
  const nonce = Buffer.from(required("ATB_PARITY_NONCE_HEX"), "hex");

  const encrypted = encryptRaw(plaintext, password, { salt, nonce });
  process.stdout.write(Buffer.from(encrypted).toString("hex"));
}

main().catch((err) => {
  console.error(err && err.stack ? err.stack : String(err));
  process.exit(1);
});
