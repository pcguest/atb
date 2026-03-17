# Post-Release Handoff Plan (v1.1.0)

## Immediate Actions (Day 0)
- [ ] Tag `v1.1.0` on `main`
- [ ] Push tag: `git push origin main --tags`
- [ ] Flip README badges after tag push:

```bash
# Run this AFTER git push origin main --tags
sed -i.bak 's|v1.1.0-rc1|v1.1.0|g' README.md
sed -i.bak 's|security-gate-pending|security-gate-passing|g' README.md
```

- [ ] Publish GitHub Release with notes from `docs/releases/v1.1.0.md`
- [ ] Verify CI/CD pipelines pass on tag
- [ ] Announce on Discord/Twitter/LinkedIn (use `.github/RELEASE_ANNOUNCEMENT.md`)

## First 48 Hours
- [ ] Monitor GitHub Issues for `v1.1.0-blocker` labels
- [ ] Respond to user feedback within 24 hours
- [ ] Verify download counts + star growth
- [ ] Check Sentry/logs for unexpected errors (if applicable)

## Week 1
- [ ] Triage all `v1.1.0` issues
- [ ] Plan v1.1.1 patch release (if critical bugs found)
- [ ] Begin v1.2.0 roadmap planning
- [ ] Update PyPI/NPM packages (if applicable)

## Support Channels
- **GitHub Issues:** https://github.com/pcguest/atb/issues
- **Discord:** TBD (Patrick to create invite before announcement)
- **Email:** TBD (Patrick to confirm public contact email before announcement)
- **Security:** [security@pcguest.dev](mailto:security@pcguest.dev)

## Rollback Plan
If critical issue found:
1. Tag `v1.1.1` with fix ASAP
2. Announce deprecation of `v1.1.0` (if severe)
3. Update README to point to `v1.1.1`

---

**Owner:** Patrick Guest (CPO)  
**Last Updated:** 2026-03-12
