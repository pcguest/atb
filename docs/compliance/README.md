# Compliance docs

These documents are for compliance engineers, auditors, legal reviewers,
and engineering teams preparing evidence for a specific review.

They explain how ATB can contribute technical evidence. They do not
produce a compliance certification, legal opinion, or conformity
assessment. ATB proves the integrity of what was recorded. It does not
by itself prove that all legally relevant events were captured.

Use the mappings like this:

1. Pick the framework or review type that matches your work, such as
   the EU AI Act, SOC 2, GDPR, or NIST AI RMF. ISO 42001 references are
   currently noted in [`docs/profiles.md`](../profiles.md) rather than a
   dedicated mapping file.
2. Find the control, article, or review question you need to answer.
3. Use the mapped ATB profile and event types to identify the evidence
   that should be present in the bundle.
4. Run `atb verify --profile <profile>` and, where needed,
   `atb trust-report --profile <profile>` to produce review material.
5. Export the bundle and reports for handoff when the reviewer needs a
   portable evidence pack.

Treat these mappings as reference guidance, not legal advice. They are a
technical starting point for explaining what ATB records and how that
recorded evidence may support a wider audit or compliance process.

Framework-specific references include the
[human-oversight mapping](./human-oversight.md), which describes supporting
evidence rather than certifying that an oversight obligation was satisfied.
