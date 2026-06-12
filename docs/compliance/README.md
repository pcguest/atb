# Compliance docs

For compliance engineers, auditors, and legal reviewers preparing technical
evidence. These pages explain how ATB **can support** a review. They do not
certify compliance, provide legal opinions, or perform conformity assessment.

## How to use these mappings

1. Identify the framework question (for example EU AI Act Article 12 logging).
2. Map it to an ATB [profile](../profiles.md) and expected event types.
3. Run `atb verify --profile <profile>` and `atb trust-report --profile <profile>`.
4. Export the bundle and reports for handoff when the reviewer needs portable evidence.

ATB proves integrity of what was recorded. It does not prove that every legally
relevant event was captured.

## Framework references

| Topic | Doc |
| --- | --- |
| EU AI Act (Article 12 logging) | [eu-ai-act.md](./eu-ai-act.md) |
| ISO/IEC 42001 control references | [profiles.md](../profiles.md) (profile obligation tables) |
| SOC 2, GDPR, NIST AI RMF, retention | Use the profile and event-type mapping workflow above; ATB supplies tamper-evident workflow evidence, not control certification |

For residual risk and CAS interpretation, see the
[auditor acceptance guide](../ciso-acceptance-guide.md).
