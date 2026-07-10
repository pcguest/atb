# Privacy reveal sidecars

ATB viewer privacy reveal operations create derived evidence. They do not
modify the authoritative `.atb` bundle being inspected.

## Ownership

ATB owns the reveal sidecar contract because sidecars are produced by
`atb view` and verified with the same ATB bundle machinery. Mortise may ingest
and retain reveal sidecars, but it must treat them as ATB-derived evidence and
must not reinterpret ATB canonicalisation or hash chaining.

## File naming

For a source bundle:

```text
run.atb/session.atb
```

the viewer writes reveal audit evidence to:

```text
run.atb/session.atb.reveals
```

The sidecar is a separate ATB bundle with its own manifest and genesis-rooted
hash chain. Its records are `privacy.reveal` events.

## `privacy.reveal` event data

New reveal records use this data contract:

| Field | Required | Meaning |
| --- | --- | --- |
| `schema_version` | Yes | `atb.reveal.sidecar.v1` |
| `seq` | Yes | Source bundle event sequence number containing the revealed field |
| `field_path` | Yes | Requested field path |
| `field_path_resolved` | No | Actual resolved field path when the request used an alias or normalized path |
| `revealed_at` | Yes | RFC 3339 UTC timestamp for the reveal |
| `user` | Yes | Viewer-authenticated or request-derived user identity available to the local server |
| `ip` | Yes | Request remote address captured by the local server |
| `reason` | No | Caller-supplied reason string |
| `source_bundle_id` | Yes when source manifest is present | Bundle ID from the authoritative source bundle manifest |
| `source_head_hash` | Yes when source bundle has records | Chain head hash of the authoritative source bundle at reveal time |

`source_bundle_id` and `source_head_hash` bind each reveal to the exact source
bundle state it annotates. A later source-bundle append creates a different
head hash; it does not rewrite existing reveal records.

## Verification behavior

Verify the authoritative bundle and sidecar separately:

```bash
atb verify --bundle run.atb/session.atb
atb verify --bundle run.atb/session.atb.reveals
```

The authoritative bundle remains the source of truth for recorded workflow
events. The reveal sidecar proves that a local viewer later revealed a field
from a specific source bundle head. It does not prove the revealed value is
complete evidence of the workflow, and it does not change profile/CAS results
for the source bundle.

## Multiple reveal sessions

Multiple reveals append to the same `.reveals` sidecar. Each record carries its
own timestamp, source sequence, field path, and source head hash. If the source
bundle changes between reveal sessions, downstream custody and retention
systems must treat records with different `source_head_hash` values as
annotations over different source states.

## Missing sidecars

If `<bundle>.reveals` is absent, ATB interprets that as "no local reveal audit
sidecar is available." It is not proof that no one ever viewed sensitive data
through another copy, viewer, export, or system.

## Custody and retention

Reveal sidecars are derived evidence and may contain sensitive audit metadata.
When retained or lodged with custody:

- store the authoritative source bundle unchanged;
- store the `.reveals` sidecar as a separate object;
- preserve the filename or metadata link to the source bundle;
- validate every sidecar `source_bundle_id` and `source_head_hash` against the
  source bundle under custody;
- apply retention at least as strict as the source bundle when the reveal
  sidecar is needed for an investigation, audit, or legal hold;
- never merge reveal records back into the source `.atb` chain.

Current ATB local compliance and incident exports do not automatically include
`.reveals` sidecars unless an operator separately collects them. Operators who
need reveal audit custody should retain both files and submit both to a custody
system that understands derived evidence.

## Compatibility

Older reveal sidecars may contain `privacy.reveal` records without
`schema_version`. Verifiers should continue to validate their ATB hash chain
but may report the sidecar contract version as unknown. New records emitted by
current ATB include `schema_version: atb.reveal.sidecar.v1`.
