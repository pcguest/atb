# BIR-001: Bundle Meta Integrity Fields

**Requestor:** UI AGENT | **Priority:** High | **Sprint:** 1

**Problem:**
Dashboard Priority 1 requires `genesis_hash` and `verified_at` in the Bundle Meta panel. These fields are not present in the current viewer API contract, so the UI cannot render them from authoritative backend data.

**Proposed Response Schema:**

```json
{
  "genesis_hash": "string (sha256 hex)",
  "verified_at": "string (RFC3339 timestamp)"
}
```

**Endpoint:** `GET /api/v1/bundle/meta` | **Backward Compatible:** Yes

**UI Dependency:**

- `web/app/view/components/bundle-meta/BundleMetaPanel.tsx`
- Required to complete Priority 1 "Bundle Meta panel: chain_length, genesis_hash (copyable), head_hash, verified_at"

**Implementation Status:**
Implemented in pkg/api/v1/handlers.go
