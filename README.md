# ATB — Agent Trace Bundle

**Tamper-evident, replayable audit trails for AI agent workflows.**

[![CI](https://github.com/pcguest/atb/actions/workflows/ci.yml/badge.svg)](https://github.com/pcguest/atb/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-indigo.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](go.mod)
[![PyPI](https://img.shields.io/pypi/v/atb-sdk)](https://pypi.org/project/atb-sdk/)
[![Python SDK](https://img.shields.io/badge/Python-3.9+-3776AB?logo=python)](sdk/python/)
[![TypeScript SDK](https://img.shields.io/badge/TypeScript-5.0+-3178C6?logo=typescript)](sdk/typescript/)

ATB is an **audit-first AI workflow runtime**. Every decision, every tool call, every output from your AI agents is recorded in a hash-chained bundle — tamper-evident, locally stored, and verifiable by anyone with the file.

---

## Why ATB?

AI agents are making consequential decisions in production. When something goes wrong — or when a regulator asks — you need a complete, verifiable record of what happened. ATB provides:

- **Tamper-evidence**: SHA-256 hash chaining means any modification to any event breaks the entire chain.
- **Replayability**: Every event is stored with its full payload, enabling exact replay and debugging.
- **Cross-language consistency**: RFC 8785 (JCS) canonicalization ensures identical hashes in Go, Python, and TypeScript.
- **Zero infrastructure**: Bundles are plain NDJSON files stored locally in `run.atb/`. No database, no server.

---

## Quick Start

### CLI (Go)

```bash
# From source
git clone https://github.com/pcguest/atb.git
cd atb
go build -o atb ./cmd/atb
./atb version

# Initialise a new bundle
./atb init

# Append events
./atb append dev.session '{"date":"2025-01-15","features":["hash chaining"]}'
./atb append decision '{"choice":"Go over Rust","reason":"Solo founder velocity"}'

# Verify integrity
./atb verify
# ✓ Bundle verified: 2 events, chain intact.

# Open a local timeline viewer
./atb view --port 8080
```

### Python SDK

```bash
# Install from PyPI (or source install for local development):
pip install atb-sdk

# Local source install:
cd sdk/python
python3 -m venv venv
source venv/bin/activate
pip install -e .
pytest -v
```

```python
from atb import Bundle

bundle = Bundle()
bundle.append("dev.session", {
    "date": "2025-01-15",
    "features_built": ["hash chaining", "CLI init"],
})
bundle.append("decision", {
    "choice": "Go over Rust for CLI",
    "reason": "Solo founder velocity",
    "alternatives": ["Rust", "Python-only"],
})
bundle.save()

# Verify integrity
b = Bundle.load()
b.verify()
print(f"✓ Verified {len(b)} events — chain intact.")
```

### TypeScript SDK

```bash
cd sdk/typescript
npm ci
npm run build
npm run typecheck
```

```typescript
import { Bundle } from "@pcguest/atb-sdk";

const bundle = new Bundle();
bundle.append("dev.session", { date: "2025-01-15" });
bundle.append("decision", { choice: "Go over Rust", reason: "velocity" });
bundle.save();

const loaded = Bundle.load();
loaded.verify();
```

### LangChain Integration

```python
from atb import Bundle
from atb.integrations.langchain import ATBCallbackHandler
from langchain.chat_models import ChatOpenAI

bundle = Bundle()
handler = ATBCallbackHandler(bundle, auto_save=True)
llm = ChatOpenAI(callbacks=[handler])
# All LLM calls are now automatically recorded.
```

---

## How It Works

Each event in an ATB bundle is a JSON record containing:

```json
{
  "event": {
    "seq": 1,
    "prev_hash": "0000000000000000000000000000000000000000000000000000000000000000",
    "type": "dev.session",
    "data": { "date": "2025-01-15", "features": ["hash chaining"] }
  },
  "hash": "cdc87dac2d8d61bf..."
}
```

The hash is computed as:

```
hash(n) = SHA256( hex(hash(n-1)) || RFC8785(event(n)) )
```

The first event uses a genesis hash of 64 zeros. This creates a cryptographic chain where any modification to any event — including reordering — produces a different hash, immediately detectable by `atb verify`.

---

## Project Structure

```
/
├── cmd/atb/              # Go CLI entry point
├── internal/
│   ├── canonicalize/     # RFC 8785 JSON canonicalization
│   ├── hash/             # SHA-256 hash chaining algorithm
│   ├── bundle/           # NDJSON bundle read/write
│   └── viewer/           # Local HTML timeline rendering
├── sdk/
│   ├── python/           # Python SDK (PyPI workflow ready)
│   └── typescript/       # TypeScript SDK (npm workflow ready)
├── web/                  # Next.js 14 landing page & platform
├── docs/                 # Specification + documentation
├── dev-log/              # ATB bundles recording ATB's own development
├── test/fixtures/        # Golden test files for cross-language verification
└── .github/workflows/    # CI/CD (ci.yml, release.yml, docs.yml)
```

---

## Event Types

ATB is schema-flexible — any JSON-serialisable payload is valid. Recommended event types:

| Type | Description |
|------|-------------|
| `dev.session` | A development session with features built and blockers |
| `decision` | An architectural or product decision with rationale |
| `release` | A version release with download statistics |
| `langchain.llm.start` | LangChain LLM invocation started |
| `langchain.chain.end` | LangChain chain completed |
| `langchain.agent.action` | LangChain agent tool selection |

---

## Roadmap

| Phase | Timeline | Status |
|-------|----------|--------|
| CLI v1.0 (Go) + Python SDK | Week 1–4 | In Progress |
| Landing Page (Next.js) | Week 1–2 | Complete |
| Documentation Site | Week 3–4 | Planned |
| Hosted Viewer (drag & drop) | Week 5–8 | Planned |
| Pro SaaS (cloud sync, sharing) | Week 9–12 | Planned |
| Enterprise (SSO, RBAC, compliance) | Week 13–16 | Planned |

---

## Contributing

Contributions are welcome. Please open an issue first to discuss what you would like to change.

1. Fork the repository.
2. Create a feature branch: `git checkout -b feat/my-feature`
3. Make your changes and add tests.
4. Run `go test ./...` and ensure all tests pass.
5. Submit a pull request using the provided template.

See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) for community expectations.

---

## License

MIT License. See [LICENSE](LICENSE) for details.

---

## Dogfooding

ATB is used to track its own development. The `dev-log/` directory contains ATB bundles recording every major development session and architectural decision. This serves as both a test of the system and a live demonstration of its value.

```bash
# Verify the bootstrap bundle
atb verify dev-log/v0.1.0-bootstrap.atb
# ✓ Bundle verified: 5 events, chain intact.
```

For an end-to-end onboarding guide, see [docs/quickstart.md](docs/quickstart.md).

## Security

- Responsible disclosure: [SECURITY.md](SECURITY.md)
- Security model and control mapping: [docs/security.md](docs/security.md)
- Incident handling: [docs/incident-response.md](docs/incident-response.md)
- Configuration and secrets: [docs/config.md](docs/config.md)
