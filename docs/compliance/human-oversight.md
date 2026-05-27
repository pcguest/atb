<!-- TODO: consolidate into docs/compliance/eu-ai-act.md -->

# Human Oversight

## Snapshot attribution

`atb snapshot` can record optional human oversight attribution for EU AI Act Article 14 evidence.
Use `--actor-id` for the human approver identity, such as an email address or user ID.
Use `--actor-role` for the approver role, such as `operator` or `auditor`.
Use `--oversight-note` for a short human rationale; notes are capped at 512 characters.
These fields are optional and support evidence capture only; obligation profiles decide when they are required.
The event type and payload context are described in [the v1 spec](../spec-v1.0.md).
The optional fields are declared in [the event schema](../../schemas/event.v1.json).
