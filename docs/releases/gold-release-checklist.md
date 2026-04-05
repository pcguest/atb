<!-- Archived release-planning draft. Not maintained. -->
# ATB Gold Release Checklist

## Pre-Release Gates
- [ ] Security sign-off (docs/security/gold-signoff.md)
- [ ] All Critical/High vulns resolved (Go + NPM)
- [ ] GitHub Actions SHA-pinned (0 version-tag pins)
- [ ] E2E tests passing (Cypress, 4/4 tests)
- [ ] Lighthouse scores met (A11y >=100, Performance >=90)
- [ ] Backend coverage >=80% (pkg/api/v1)
- [ ] UI coverage >=80% (web/components)

## Documentation
- [ ] Release notes updated in `docs/releases/`
- [ ] Getting started guide validated in `docs/guides/`
- [ ] API docs synced (docs/api/openapi.yaml regenerated)
- [ ] SECURITY.md updated with known issues
- [ ] CHANGELOG.md updated with all changes since the previous beta release
- [ ] README.md badges updated (version, security, coverage)

## Build & Distribution
- [ ] Go binary builds cleanly (go build -o atb ./cmd/atb)
- [ ] UI builds cleanly (cd web && npm run build)
- [ ] Embed test passes (make test-embed)
- [ ] Docker image builds (if applicable)
- [ ] Binary size <20% growth from the previous beta snapshot

## Testing
- [ ] All unit tests pass (go test ./...)
- [ ] All E2E tests pass (make test-e2e)
- [ ] Performance tests pass (make test-performance)
- [ ] Manual QA pass (see QA checklist below)

## Manual QA Checklist
- [ ] `atb init` creates bundle successfully
- [ ] `atb append` adds events to chain
- [ ] `atb verify` validates chain integrity
- [ ] `atb view --ui-experimental` loads dashboard
- [ ] Trust Score displays and updates
- [ ] Role switching works (Engineer/Auditor/Executive)
- [ ] Privacy reveal requires auth (401 without token)
- [ ] Rate limiting works (429 on 11th request)
- [ ] CSP headers present in browser devtools
- [ ] Export generates valid ZIP with manifest

## Post-Release
- [ ] GitHub Release published with notes
- [ ] Git tag pushed (current beta tag)
- [ ] PyPI package updated (if applicable)
- [ ] NPM package updated (if applicable)
- [ ] Docker Hub image pushed (if applicable)
- [ ] Announcement posted (Discord, Twitter, LinkedIn)
- [ ] Internal stakeholders notified

## Rollback Plan
If critical issues found post-release:
1. Document the issue in GitHub Issues with a `release-blocker` label
2. Create hotfix branch: `hotfix/current-beta-<issue-name>`
3. Fix, test, and retag the current beta patch
4. Communicate to users via release notes + announcement

---

**Sign-Off Required:**
- [ ] CPO (Patrick Guest)
- [ ] Security review
- [ ] Release validation
- [ ] Product QA
