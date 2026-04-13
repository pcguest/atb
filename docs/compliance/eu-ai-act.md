# EU AI Act Article 12 and ATB

> **Note:** This document is based on technical analysis of Article 12 of Regulation (EU) 2024/1689 as of April 2026 and has not been reviewed by legal counsel. Do not cite it in regulatory submissions without independent legal review.

## Overview

Article 12 of Regulation (EU) 2024/1689 requires providers of high-risk AI systems to ensure that their systems are technically capable of automatically recording events (logs) over the lifetime of the system, with logging appropriate to the intended purpose and sufficient to ensure traceability. Broad application of many obligations under the Act, including for providers placing high-risk systems on the market, begins on 2 August 2026 (subject to sector-specific and transitional rules elsewhere in the Regulation).

ATB contributes tamper-evident integrity for whatever events are written into a bundle, and profile-scoped completeness assurance scoring (CAS) against declared obligation templates. It does not by itself demonstrate that every legally material event in a real-world workflow was captured, or that operators configured instrumentation exhaustively.

## Profile mapping

| Profile ID | Workflow class | Article 12 traceability addressed | Gap / out of scope |
| --- | --- | --- | --- |
| `atb.profile.rag_answer` | `rag_answer` | Traceability of model invocation, retrieval basis, and policy application for advisory AI outputs when those events are recorded. | This does not prove retrieval was complete or that the model provider executed inference exactly as recorded. |
| `atb.profile.privileged_tool_action` | `privileged_tool_action` | Traceability of authorisation, execution, and commitment for high-impact actions when gated events are present in the bundle. | Actions executed outside the ATB gate are invisible, and tool provider internals are not attested. |
| `atb.profile.data_export` | `data_export` | Traceability of export authorisation, approval, and data movement when the export control-plane events are recorded. | Downstream recipient handling after delivery is out of scope. |
| `atb.profile.policy_decision` | `policy_decision` | Traceability of which policy version was applied to which subject and action when policy and request events are recorded. | This does not prove the policy text itself was correct or that all applicable policies were considered. |
| `atb.profile.human_override` | `human_override` | Traceability of human approval identity and justification for overridden actions when approval and action events are recorded. | Physical-world identity is not verified beyond the IdP assertion, and approver comprehension of context is not attested. |
| `atb.profile.background_automation` | `background_automation` | Traceability of job scheduling, execution steps, and outputs when job lifecycle events are recorded. | Steps inside the worker that are not explicitly logged are invisible, and external side effects are out of scope. |

## What ATB does not satisfy alone

ATB does not provide: (a) a certified WORM storage path for the retention period Article 12 implies (operators must arrange durable immutable storage independently); (b) proof of capture completeness (CAS is a bounded proxy over recorded evidence, not a universal guarantee); (c) legal advice on whether a given system is high-risk under the Act or how Article 12 applies to a specific deployment.
