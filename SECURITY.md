# Security Policy

## Supported Versions

Security fixes are shipped on the current release tag only.

| Version | Supported |
| --- | --- |
| `v1.7.0` | Yes |
| Older releases | No |

## Reporting a Vulnerability

Please report vulnerabilities privately to
[patrickcguest@proton.me](mailto:patrickcguest@proton.me).

Include:

- A clear description of the issue and impact.
- Reproduction steps or proof of concept.
- Affected versions/commits and environment details.

Do not open public GitHub issues for unpatched vulnerabilities.

## Disclosure Process

- Initial acknowledgment target: within 5 business days.
- Triage and remediation plan: after issue reproduction.
- Coordinated disclosure: after a fix is available.

## Scope

This policy applies to:

- Go CLI (`cmd/`, `internal/`)
- SDKs (`sdk/python`, `sdk/typescript`)
- Dashboard (`web/`)
- CI/CD and release workflows (`.github/workflows/`)

Trivy vulnerability scanning runs on a weekly schedule via GitHub Actions.

## Safe Harbor

Good-faith research is authorized when you avoid privacy violations, data loss, service disruption, and social engineering.
