# Why ATB

## The problem: logging vs proof

Most AI systems already produce logs. They emit request IDs, timestamps, prompts, tool calls, policy checks, and model outputs. That data is useful for debugging, but usefulness is not the same as proof. Conventional logs are easy to mutate after the fact or re-export in a different order. When a team needs to answer "can we show that this record was not altered after capture?", ordinary logging stops short.

ATB is designed for that harder question. It records events into a local bundle where each event is linked to the previous one by a SHA-256 hash chain over RFC 8785 canonical JSON. If any event is changed, removed, or reordered, verification fails. The resulting bundle is portable and can be checked without depending on a third-party service.

That is the distinction ATB cares about: logging tells you what a system says happened; proof tells you whether the recorded artefact still matches what was captured.

## What integrity means and what ATB proves

In ATB, integrity means that the recorded bundle is tamper-evident. Each record carries the previous hash and its own content is canonicalised before hashing, so the chain is deterministic across environments. New bundles begin with a manifest record, and event timestamps are part of the integrity-protected event data rather than loose metadata beside it.

The practical artefact is the bundle itself plus the ability to run `atb verify`. A verifier needs the bundle and the ATB verification logic. If `atb verify` passes, the claim is narrow but strong: the recorded sequence of events has not been altered since capture.

That matters in incident review, internal audit, customer handoff, and privacy-sensitive AI operations. Teams can preserve a concrete evidence trail and verify it later without reconstructing the narrative from screenshots or mutable dashboards.

## What completeness means and what ATB does not claim

Completeness is a different property. It asks whether everything relevant was recorded in the first place. A complete audit system would need confidence that all required execution paths were instrumented, that no important events were skipped, that payloads were captured before material transformation, and that timing reflects what actually happened.

ATB does not claim that. It does not prove that every relevant event entered the chain. It does not prove that instrumentation existed on every branch, that a developer did not log only selected paths, or that an upstream component did not reshape data before handing it to ATB.

A system can honestly prove integrity today without pretending it has solved completeness. That makes the evidence claim narrower, but more defensible.

## The five failure modes completeness must address

First, omission: a relevant event never enters the audit trail at all.

Second, selective capture: the system logs some paths but not others, often because instrumentation is uneven across success, failure, retry, or fallback flows.

Third, semantic reshaping: the payload that reaches the logger is already transformed, redacted, summarised, or normalised in a way that hides what actually drove the decision.

Fourth, timing manipulation: events are recorded with misleading timestamps, delayed writes, or reconstructed order, so the resulting trail looks coherent but does not faithfully represent sequence.

Fifth, path skipping: entire code paths or execution environments sit outside the instrumentation boundary, so meaningful behaviour never becomes auditable evidence.

These are real problems, but they are not solved by stronger hashing alone. They require instrumentation discipline, coverage analysis, minimum evidence expectations, and sometimes external controls around time and execution.

## How integrity and coverage fit together in practice

For teams using ATB well, integrity and coverage are complementary rather than competing ideas. ATB should be treated as the integrity layer for the evidence you do capture. Coverage work then determines how much of the underlying system is represented in that evidence.

In practice, that means defining which events must exist for a given workflow, instrumenting them consistently, and treating missing expected records as an operational signal. The hash chain makes the captured evidence durable and independently verifiable. Coverage work makes that evidence representative enough to support governance, review, and incident reconstruction.

This division is useful because it keeps system design honest. It avoids overstating what the tooling proves while still producing a concrete artefact that can survive scrutiny. For many teams, the right order is to make recorded evidence tamper-evident first, then improve how much of the workflow enters that evidence.

## What this means for teams using ATB today

Teams using ATB today should treat a verified bundle as trustworthy evidence of recorded events, not as proof that nothing relevant was left out. It means incident review can rely on a stable artefact, customer handoff can include something more rigorous than exported screenshots, and internal audit or privacy review can inspect a bundle and run `atb verify` independently.

The discipline this requires is straightforward. Be explicit about what your instrumentation covers. Define expected event types for important workflows. Review missing events as coverage defects. Keep the integrity claim narrow and precise: ATB proves that what was recorded was not altered after capture. It does not claim that the recording boundary was complete.

That distinction strengthens the product position. A narrow claim that survives inspection is more useful than a broad claim that cannot be defended. ATB gives teams an independently verifiable local evidence format today, while leaving room for stronger coverage and traceability controls around it.
