# ATB v1.0 Launch Checklist

## Current Status (March 3, 2026)

- [x] Git author attribution standardized (`Paddy Guest <patrickcguest@proton.me>`)
- [x] TypeScript SDK lockfile added (`sdk/typescript/package-lock.json`)
- [x] TypeScript SDK typecheck passes locally
- [x] Python SDK placeholder tests present and passing locally (`3 passed`)
- [x] CI Discord step guarded when `DISCORD_WEBHOOK_URL` is missing
- [x] ATB dogfooding trace recorded in `run.atb/`
- [x] README install/test instructions updated for current state

## Manual External Steps

- [x] Push local commits to `main`
- [x] Confirm GitHub Actions is green on `main`
- [ ] Add `DISCORD_WEBHOOK_URL` repository secret (if notifications desired)
- [ ] Deploy landing page to Vercel (`web/` root)
- [ ] Add deployed URL + waitlist URL to README

## Launch Day

- [ ] Post to Hacker News
- [ ] Post to Twitter/X
- [ ] Post to r/MachineLearning
- [ ] DM 5 potential early users
- [ ] Monitor GitHub Issues and Discord for first 24 hours

## Post-Launch: Feedback Loop

- [ ] Enable issue auto-labeling workflow
- [ ] Enable weekly feedback digest workflow
- [ ] Send first 3 early-user DMs (use `docs/feedback/dm-template.md`)
- [ ] Track blockers in Discord (`impact:blocker`)
- [ ] Review Monday digest and prioritize top 3 asks

## Week 2 Priorities

- [x] Add `atb view` local HTML viewer command
- [ ] Publish Python SDK to PyPI (workflow ready; first tag pending)
- [x] Publish quickstart tutorial (`docs/quickstart.md`)
- [ ] Publish TypeScript SDK to npm
- [x] Draft `atb push` v1.1 cloud-sharing spec (`docs/spec/atb-push-v1.1.md`)
- [ ] Ship first pro-tier onboarding flow
