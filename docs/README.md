# ATB documentation

This index lists the surviving docs files and populated subdirectories.

## Core specs

- [architecture.md](./architecture.md) - Repository architecture and package layout.
- [config.md](./config.md) - CLI configuration and environment settings.
- [security.md](./security.md) - Security model, threat boundaries, and response links.
- [key-management.md](./key-management.md) - Key handling and encryption guidance.
- [performance.md](./performance.md) - Current performance baseline and measurement notes.
- [spec-v1.0.md](./spec-v1.0.md) - Frozen v1.0 bundle format specification.
- [spec-ai-traces.md](./spec-ai-traces.md) - AI trace event format and field rules.
- [spec-dashboard.md](./spec-dashboard.md) - Local dashboard behaviour, routes, and safety rules.
- [api/](./api/) - Local viewer API reference files.
- [api/README.md](./api/README.md) - API doc regeneration steps.
- [api/openapi.yaml](./api/openapi.yaml) - OpenAPI description for the local viewer API.
- [api/verify-schema.md](./api/verify-schema.md) - JSON schema notes for `atb verify --format json`.
- [spec/](./spec/) - Draft and in-progress specifications outside the frozen v1.0 spec.
- [spec/bundle-push.md](./spec/bundle-push.md) - In-progress specification for explicit bundle push export.

## Profiles

- [profiles.md](./profiles.md) - Canonical obligation profiles and CAS scoring notes.

## Integrations

- [ai-integration.md](./ai-integration.md) - Integration overview for recording ATB events from AI systems.
- [integrations/](./integrations/) - Integration-specific guides and export notes.
- [integrations/README.md](./integrations/README.md) - Index of shipped integration guides.
- [integrations/langchain.md](./integrations/langchain.md) - LangChain recording guidance for Python workflows.
- [integrations/vercel-ai.md](./integrations/vercel-ai.md) - Vercel AI SDK recording guidance for TypeScript workflows.
- [integrations/mcp.md](./integrations/mcp.md) - MCP bridge notes and current scope.
- [integrations/siem-grc.md](./integrations/siem-grc.md) - Export patterns for SIEM and GRC review flows.
- [integrations/worm-s3.md](./integrations/worm-s3.md) - Explicit WORM/S3 export behaviour and limits.

## Guides

- [quickstart.md](./quickstart.md) - Short CLI path for creating, verifying, and exporting a bundle.
- [why-atb.md](./why-atb.md) - Product scope, category boundaries, and intended use.
- [incident-response.md](./incident-response.md) - Maintainer incident response runbook.
- [getting-started/](./getting-started/) - Introductory path for new users.
- [getting-started/README.md](./getting-started/README.md) - First-run commands and recommended reading order.
- [guides/](./guides/) - Operational and workflow guides.
- [guides/README.md](./guides/README.md) - Index of practical workflow and operations guides.
- [guides/incident-review-workflow.md](./guides/incident-review-workflow.md) - Local incident review workflow and expected outputs.
- [guides/customer-handoff-workflow.md](./guides/customer-handoff-workflow.md) - Customer handoff workflow for encrypted or exported bundles.
- [comparisons/](./comparisons/) - Comparisons against hosted tooling and ad hoc evidence.
- [comparisons/README.md](./comparisons/README.md) - Index of comparison documents.
- [comparisons/hosted-observability.md](./comparisons/hosted-observability.md) - ATB boundary against hosted AI observability products.
- [comparisons/logs-and-screenshots.md](./comparisons/logs-and-screenshots.md) - Comparison against manual evidence collection.
- [comparisons/opentelemetry.md](./comparisons/opentelemetry.md) - Relationship between ATB and OpenTelemetry.
- [use-cases/](./use-cases/) - Scenario-specific notes for review and handoff workflows.
- [use-cases/README.md](./use-cases/README.md) - Index of current use-case documents.
- [use-cases/incident-review.md](./use-cases/incident-review.md) - Incident review use case for private AI workflows.
- [use-cases/customer-handoff.md](./use-cases/customer-handoff.md) - Customer handoff use case without hosted trace dependence.
- [use-cases/internal-audit-privacy-review.md](./use-cases/internal-audit-privacy-review.md) - Internal audit and privacy review use case.
- [maintenance/](./maintenance/) - Maintenance and recovery procedures.
- [maintenance/disaster-recovery.md](./maintenance/disaster-recovery.md) - Quarterly disaster recovery test procedure.

## Compliance

- [ciso-acceptance-guide.md](./ciso-acceptance-guide.md) - Security review questions, claims, and limits.
- [compliance/](./compliance/) - Export references, mappings, and static support files.
- [compliance/export.md](./compliance/export.md) - Compliance export formats and contents.
- [compliance/eu-ai-act.md](./compliance/eu-ai-act.md) - Article 12 mapping and evidence limits.
- [compliance/nist-ai-rmf.md](./compliance/nist-ai-rmf.md) - NIST AI RMF mapping notes.
- [compliance/soc2.md](./compliance/soc2.md) - SOC 2 evidence bundle contents.
- [compliance/gdpr.md](./compliance/gdpr.md) - GDPR export behaviour for DSR and records support.
- [compliance/retention.md](./compliance/retention.md) - Retention guidance and export handling.
- [compliance/pii-fields.json](./compliance/pii-fields.json) - Sample PII field configuration used by export flows.

## Release ops

- [contributing.md](./contributing.md) - Contributor workflow and repository hygiene checks.
- [roadmap.md](./roadmap.md) - Planned work and current status notes.
- [launch/](./launch/) - Public release communication templates.
- [launch/README.md](./launch/README.md) - Release communication template index.
- [launch/release-announcement-template.md](./launch/release-announcement-template.md) - Public release announcement template.
- [launch/release-notes-template.md](./launch/release-notes-template.md) - Internal release notes drafting template.
- [release/](./release/) - Maintainer release operation references.
- [release/README.md](./release/README.md) - Release operations index.
- [release/secrets.md](./release/secrets.md) - Names-only inventory of release secrets.
