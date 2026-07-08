# Authentication and role boundaries

ATB and Mortise are separate systems with separate authentication models.

## ATB viewer

`atb view` is loopback-first and protects its API with a generated session
token. It can also validate OIDC/JWT when `--oidc-issuer` and
`--oidc-audience` are configured. See `atb view --help` for the current flags.

Viewer roles are:

- `viewer` and `auditor`: read-only review APIs.
- `operator` and `admin`: mutating actions such as privacy reveal.

ATB remains a local evidence tool; it is not an identity provider or hosted
multi-tenant access-control service.

## Mortise

Mortise authentication is configured in the separate Mortise repository.
Current supported modes are:

- `MORTISE_AUTH_TOKEN` for a single deployment token.
- `--api-keys-file` for hashed API-key lookup, organisation isolation,
  retention policy, and `admin` versus read-only `auditor` roles.

The daemon rejects unauthenticated non-loopback binds. Production deployments
must still terminate TLS at a reverse proxy, rotate credentials, enforce
tenant-specific quotas, and preferably add short-lived OIDC or mTLS identity at
that edge.

ATB clients authenticate to Mortise with `ATB_MORTISE_TOKEN`. The token is read
from the environment rather than a CLI flag.

See the Mortise
[runbook](https://github.com/pcguest/mortise/blob/main/docs/runbook.md) and
[threat model](https://github.com/pcguest/mortise/blob/main/docs/mortise-threat-model.md)
for the authoritative production controls.
