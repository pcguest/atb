# ATB — Agent Trace Bundle

[![Tests](https://github.com/pcguest/atb/actions/workflows/ci.yml/badge.svg)](https://github.com/pcguest/atb/actions/workflows/ci.yml)
[![Security Gate (HIGH/CRITICAL=0)](https://github.com/pcguest/atb/actions/workflows/security.yml/badge.svg)](https://github.com/pcguest/atb/actions/workflows/security.yml)
[![PyPI](https://img.shields.io/pypi/v/atb-sdk?label=PyPI)](https://pypi.org/project/atb-sdk/)
[![npm](https://img.shields.io/npm/v/%40pcguest%2Fatb-sdk?label=npm)](https://www.npmjs.com/package/@pcguest/atb-sdk)
[![License: MIT](https://img.shields.io/badge/License-MIT-indigo.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](go.mod)

> Cryptographically verifiable audit trails for AI agent workflows.

## Why ATB?

- **Zero-knowledge by default:** secrets and keys remain client-side.
- **Tamper-evident by design:** SHA-256 hash chaining with RFC 8785 canonicalization.
- **Local-first operation:** works offline with plain files; cloud is optional.

---

## Try It in 60 Seconds

```bash
pip install atb-sdk
git clone https://github.com/pcguest/atb.git
cd atb
GOBIN="$PWD/.bin" go install ./cmd/atb
export PATH="$PWD/.bin:$PATH"

atb init
atb append agent.step --data '{"input":"hello","output":"world"}'
atb verify
```

## Learn More

- [Quickstart](docs/quickstart.md)
- [Security Model](docs/security.md)
- [AI Integration](docs/ai-integration.md)
- [Specification](docs/spec-v1.0.md)

---

## Developer Setup

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
