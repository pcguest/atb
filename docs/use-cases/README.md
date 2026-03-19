# ATB Use Cases

ATB is most useful when teams need a verifiable record of AI behaviour without default external trace storage.

## Current Use Cases

- [Incident Review for Private AI Workflows](./incident-review.md)
- [Customer Handoff Without Platform Lock-In](./customer-handoff.md)
- [Internal Audit and Privacy Review on a Local Bundle](./internal-audit-privacy-review.md)

## Fit Check

ATB is a strong fit when:

- traces contain sensitive internal or customer data
- security, legal, or compliance teams need to inspect what happened
- teams want a portable bundle they can verify locally

ATB is a weak fit when:

- the main requirement is collaborative cloud debugging
- the team needs prompt management, eval suites, or hosted workspaces first
- local storage and cryptographic verification are not meaningful constraints
