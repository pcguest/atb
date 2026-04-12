# Incident Response Runbook

This runbook is intentionally lightweight so it remains usable by a solo maintainer.

## 1. Detection and Triage

Trigger conditions:

- Report received through `patrickcguest@proton.me`
- Suspicious release artefact behaviour
- Integrity mismatch or unexpected production workflow failures
- Credential leak signal (logs, GitHub alert, third-party report)

Immediate actions:

- Open a private incident note with timestamped events.
- Assign a severity:
  - Sev 1: active compromise or confirmed data exposure
  - Sev 2: exploitable vulnerability, no known exploitation
  - Sev 3: low-impact weakness or hardening gap

## 2. Containment

- Revoke and rotate affected credentials:
  - `NPM_TOKEN`
  - `DOCKERHUB_TOKEN`
  - `DISCORD_WEBHOOK_URL`
- Review GitHub OIDC trusted publishing configuration for PyPI release access.
- Pause release/publish workflows if needed.
- Disable any compromised integration endpoints.

## 3. Eradication and Recovery

- Patch root cause on `main`.
- Add or update tests to prevent regression.
- Rebuild and re-verify release artefacts.
- Re-enable workflows only after checks are green.

## 4. Communication

- Acknowledge security reports within 5 business days.
- For user-impacting incidents, publish a concise postmortem summary:
  - What happened
  - What was affected
  - What was done to fix it
  - What changed to prevent recurrence

## 5. Evidence and Follow-up

- Preserve relevant CI logs, commit references, and release metadata.
- Record timeline, impact, and remediation in a postmortem document.
- Convert follow-up gaps into actionable issues with owners and due dates.

## Contacts

- Security intake: [patrickcguest@proton.me](mailto:patrickcguest@proton.me)
- Maintainer: [patrickcguest@proton.me](mailto:patrickcguest@proton.me)
