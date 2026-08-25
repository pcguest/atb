# Security scanner suppressions

This register covers every first-party `#nosec` directive accepted by the Go
security gate. Each directive must name the gosec rule and include its local
reason on the same line. New blanket exclusions are not permitted.

The v1.15.3 audit contains 64 rule suppressions across 36 files. Gosec v2.27.1
reports zero unsuppressed findings. Reconcile the register with:

```bash
rg -n '#nosec|nolint:gosec' --glob '*.go' .
gosec $(go list -f '{{.Dir}}' ./... | grep -v '/node_modules/')
```

## Accepted exception classes

| Rules | Scope | Why the exception is valid | Required control |
| --- | --- | --- | --- |
| `G304`, `G703` | Local bundle, config, key, ledger, archive, export, and identity paths | Reading or writing the operator-selected local path is the command's explicit contract; it is not a remotely supplied server path. | Paths are cleaned or derived from fixed names, sensitive output uses owner-only modes, and callers validate bundle/config semantics. |
| `G301`, `G306` | Directory and file creation | Gosec's generic mode recommendation is stricter or mismatched for these local artefacts. | Directories use `0750`; private/config/bundle files use `0600`. |
| `G107` | RFC 3161 timestamp endpoint | Remote TSA access is an explicit feature. | URL validation permits only credential-free HTTP(S), and response time/size are bounded. |
| `G204` | `capture run` and local browser launch | Executing the operator's command is the purpose of `capture run`; viewer launch selects a fixed platform binary. | Capture arguments are passed without a shell; browser URLs are restricted to loopback HTTP(S). |
| `G112` | Local agent and viewer servers | The flagged server literals already set `ReadHeaderTimeout`. | Both services are loopback-only and configure a five-second header timeout. |
| `G124` | Viewer reveal cookie | `Secure=true` would make the intended plain-HTTP loopback viewer unusable. | `Secure` follows TLS state; the cookie remains `HttpOnly` and `SameSite=Strict`, behind session authentication. |

## Locations

- CLI boundary: `cmd/atb/{anchor,archive,capture,compliance,config,decrypt,encrypt,export,incident,keygen,main,push,sign,verify,view}.go`
- Bundle and storage boundary: `internal/{anchor,archive,bundle,compliancepack,incident,push,verify}`
- Local services and identity: `internal/{agent,identity,proxy}`
- Public custody handoff: `pkg/custody/export.go`
- Tests: `cmd/atb/{customer_handoff_workflow,lock_contention_unix}_test.go`

The inline directive is the authoritative per-occurrence entry because it
travels with the code and states the exact data-flow control. This document is
the reviewed category register and must be updated when a new rule or boundary
class is introduced.
