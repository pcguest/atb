# ATB SDKs

## Overview

The SDKs provide the open read layer for ATB bundles: loading bundle files, iterating events, and checking local hash-chain integrity from application code.

## Python SDK

The [Python SDK](python/) exposes `Bundle`, event helpers, hash parity utilities,
and capture integrations. It creates, reads, and verifies local bundle chains;
the Go CLI adds profile, incident, anchor, and full report evaluation.

## TypeScript SDK

The [TypeScript SDK](typescript/) exposes the same creation, reader, and local
verification surface for Node.js projects, with typed records and event
constants for application integrations.
