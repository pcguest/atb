# ATB Viewer API Contract

This document mirrors the current backend payloads used by the Trust Dashboard shell.

## Endpoints Used

### `GET /api/v1/verification`

- Required fields:
  - `status`: `"valid" | "invalid"`
  - `message`: `string`
  - `bundle_path`: `string`
  - `chain_length`: `number`
  - `head_hash?`: `string`
- UI dependency:
  - Verification status badge
  - Trust score weighting
  - Bundle Meta chain length/head hash

### `GET /api/v1/bundle/meta`

- Required fields:
  - `bundle_path`: `string`
  - `event_count`: `number`
  - `type_counts`: `Record<string, number>`
  - `first_timestamp?`: `string`
  - `last_timestamp?`: `string`
  - `verified`: `boolean`
  - `verification_message`: `string`
- UI dependency:
  - Trust score freshness signal (`last_timestamp`)
  - Bundle path context
  - Verification summary

### `GET /api/v1/bundle/events`

- Required fields:
  - `offset`, `limit`, `total`: `number`
  - `events[]`: event payload objects
- UI dependency:
  - Engineer-only timeline and inspector

### `GET /api/v1/bundle/graph`

- Required fields:
  - `nodes[]`, `edges[]`
- UI dependency:
  - Engineer-only trace graph

### `POST /api/v1/privacy/reveal`

- Request fields:
  - `seq`: `number`
  - `field_path`: `string`
  - `reason?`: `string`
- Response fields:
  - `seq`, `field_path`, `value`
- UI dependency:
  - Engineer-only reveal action in event inspector

## Validation Rule

All endpoint responses are treated as untrusted and must pass Zod schemas in `web/lib/schemas/` before use.
