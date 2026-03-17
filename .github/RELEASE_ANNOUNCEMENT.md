# 🎉 ATB v1.1.0: Trust Dashboard — Gold Release

**Release Date:** 2026-03-12  
**Tag:** `v1.1.0`  
**Download:** https://github.com/pcguest/atb/releases/tag/v1.1.0

---

## What's New

ATB v1.1.0 transforms the basic audit trail viewer into an **enterprise-grade Trust Dashboard** — making cryptographic verification accessible, visual, and actionable.

### 🎨 Trust Dashboard UI
- **Built with:** Next.js 14 + React 18 + TypeScript
- **Design System:** shadcn/ui + Tailwind CSS + Recharts
- **Accessibility:** WCAG 2.1 AA compliant (100/100 Lighthouse score)

### 🔐 Security Enhancements
- Token-based auth for sensitive operations
- Rate limiting to prevent enumeration attacks
- Hash-chained audit logging (tamper-evident)
- Strict CSP headers on embedded UI
- Configurable PII masking

### 👥 Role-Based Views
| Role | Capabilities |
|------|-------------|
| **Engineer** | Raw payloads, debug info, full hash chain |
| **Auditor** | Compliance stats, evidence export, no raw PII |
| **Executive** | Trust Score, high-level trends, summary only |

### 📊 Trust Score
A 0-100 algorithmic metric based on:
- Chain continuity (40%)
- Encryption status (30%)
- Timestamp consistency (20%)
- Export freshness (10%)

---

## Quick Start

```bash
# Clone and build
git clone https://github.com/pcguest/atb.git
cd atb
git checkout v1.1.0
go build -o atb ./cmd/atb

# Initialize and run
./atb init
./atb append --type=milestone --data='{"name":"Project Started"}'
./atb view --ui-experimental
```

---

## Migration from v1.0.3

**Breaking Changes:** None — v1.1.0 is backward compatible.

**Recommended Steps:**
1. Update Go module: `go get github.com/pcguest/atb@v1.1.0`
2. Update NPM package: `npm update @pcguest/atb-sdk` (if applicable)
3. Rebuild UI: `cd web && npm install && npm run build`
4. Update config: Add `reveal_rate_limit` to `.atb/config.json` (optional)

---

## Testing

```bash
# Run all tests
make test-all

# Run E2E tests
cd web && npm run test:e2e

# Run security scans
make security-scan
```

---

## Known Issues

| Issue | Severity | Workaround | Planned Fix |
|-------|----------|------------|-------------|
| G304 gosec in export.go | Low | File paths are trusted | v1.2.0 refactor |
| Lighthouse requires Chrome | Medium | Use Docker runner | v1.1.1 Puppeteer |
| Cypress unstable in some envs | Medium | Use Docker runner | v1.1.1 stabilization |

---

## What's Next (v1.2.0 Roadmap)

- [ ] Cloud sync option (hybrid privacy model)
- [ ] WebSocket real-time streaming (replace polling)
- [ ] Performance optimization for 10k+ block bundles
- [ ] Plugin system for custom UI components
- [ ] Multi-bundle management (workspaces)

---

## Acknowledgments

Built by **Patrick Guest** (2nd Year IT Developer) with assistance from:
- Security Agent (audit + validation)
- Hygiene Agent (CI/CD + quality gates)
- UI Agent (React dashboard + accessibility)
- Context Agent (documentation + release management)

**Thank you** to the early testers and contributors who made this release possible.

---

## Get Involved

- 📚 **Docs:** https://github.com/pcguest/atb/tree/main/docs
- 🐛 **Issues:** https://github.com/pcguest/atb/issues
- 💬 **Discord:** [invite link]
- 🐦 **Twitter:** [@pcguest](https://twitter.com/pcguest)

---

**Security Notice:** If you find a vulnerability, please report it privately to [security@pcguest.dev](mailto:security@pcguest.dev) before public disclosure.

---

*ATB — The Gold Standard for AI Transparency* 🔐✨
