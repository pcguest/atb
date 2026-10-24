# ATB documentation

Start with the project [README](../README.md) and follow the ordered route below.
The canonical explanation of ATB’s architecture and limits is in
[concepts and architecture](concepts/architecture.md).

## Start

[Quickstart](getting-started/quickstart.md) → [concepts and architecture](concepts/architecture.md) → [verification and limitations](verification-meaning.md) → [integrations](integrations/README.md) → [contributing](../CONTRIBUTING.md).

- [Five-minute quickstart](getting-started/quickstart.md)
- [Examples](../examples/README.md)
- [Capture overview](capture/overview.md)
- [SDKs](../sdk/README.md)

## Verify and investigate

- [Offline verification](investigate/verify.md)
- [Incident reconstruction](investigate/incidents.md)
- [Tampering demonstration](investigate/tampering.md)
- [Trust model and limits](concepts/trust-model.md)

## Integrate

- [Integration index](integrations/README.md)
- [WORM/S3 bundle push](specification/bundle-push.md)
- [MCP](integrations/mcp.md)
- [Optional custody handoff](integrations/mortise.md)

Integrations are optional. ATB can capture, verify, and inspect bundles
without a hosted ATB service or an external custody product.

## Reference

- [Architecture](concepts/architecture.md)
- [Evidence model](concepts/evidence-model.md)
- [Glossary](concepts/glossary.md)
- [Bundle format and hashing](specification/bundle-v1.md)
- [Event semantics](specification/events.md)
- [Evidence attribution](specification/attribution.md)
- [Profiles and assurance scoring](evidence/profiles.md)
- [CAS](evidence/cas.md)
- [Key management](evidence/key-management.md)
- [Verifier report](specification/verify-report.md)
- [Local viewer API](api/README.md)

## Maintain

- [Contributing](../CONTRIBUTING.md)
- [Security policy](../SECURITY.md)
- [Versioning](../VERSIONING.md)
- [Release runbook](maintainers/release.md)
- [Support matrix](maintainers/support-matrix.md)
- [Roadmap](roadmap.md)

Compliance mappings are technical references, not legal advice, certification,
or conformity assessments. See the [compliance index](compliance/README.md).
