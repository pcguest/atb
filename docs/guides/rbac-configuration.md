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

Mortise authentication is configured independently of ATB. Its supported
identity, organisation, network, and deployment controls are defined by the
specific Mortise release, not by the ATB viewer model.

ATB clients authenticate to Mortise with `ATB_MORTISE_TOKEN`. The token is read
from the environment rather than a CLI flag.

Consult the documentation for the deployed Mortise release for its
authoritative production controls. ATB does not share viewer sessions or
identity state with Mortise.
