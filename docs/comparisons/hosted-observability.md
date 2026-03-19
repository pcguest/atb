# ATB vs Hosted AI Observability

ATB should not be evaluated as a cheaper hosted observability platform.

Hosted AI observability tools and ATB solve adjacent problems, but they optimise for different constraints.

## Category Difference

Hosted platforms are usually optimised for:

- centralised dashboards across many services and teams
- shared workspaces, collaboration, and online access
- prompt management, evals, and broader production operations

ATB is optimised for:

- local-first trace storage and verification
- tamper-evident bundles that can be checked independently
- privacy-sensitive review workflows where raw traces should not default to external storage
- deterministic evidence exports for customer, audit, or incident workflows

## Important Boundary

Hosted and self-hosted observability platforms have become better at compliance packaging, identity controls, and enterprise deployment. That is real progress in the category.

ATB should not be positioned as if it wins simply because it is "more private" or "self-hosted". The stronger distinction is different:

- hosted platforms give you a control plane
- ATB gives you a portable, verifiable evidence artefact

## Practical Comparison

| Question | Hosted AI observability | ATB |
| --- | --- | --- |
| Where do traces live by default? | Vendor-managed or centrally hosted storage | Local bundle files |
| What is the primary value? | Team-wide operational visibility | Verifiable local evidence |
| What happens after an incident? | Review logs in the platform | Verify the bundle, inspect locally, export evidence |
| How does collaboration work today? | Shared dashboards and workspaces | Portable local artefacts, no hosted workspace |
| Is local control part of the product shape? | Sometimes optional | Yes, by design |

## When ATB Is The Better Fit

Use ATB first when:

- security or customer policy makes hosted trace storage difficult
- the review process depends on proving integrity, not just observing behaviour
- you need a portable artefact for handoff or formal review
- you do not want post-incident review to depend on continued access to a vendor control plane

## When A Hosted Platform Is The Better Fit

Use a hosted platform first when:

- the main problem is shared debugging across a large team
- you need managed collaboration, evals, or prompt tooling
- the team is comfortable centralising trace storage in a vendor platform

## Current Product Boundary

ATB does not currently claim:

- hosted workspaces
- collaborative review queues
- seat-based admin controls
- managed cloud retention

That boundary is intentional. ATB is focused on giving privacy-sensitive teams a verifiable audit trail, not a generic control plane.
