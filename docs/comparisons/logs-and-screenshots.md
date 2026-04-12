# ATB vs Logs, Screenshots, and Ad Hoc Exports

Many teams do not start by comparing ATB to a vendor. They start by comparing it to what they already do now: logs, screenshots, copied JSON, and manually assembled review packs.

That is the more practical comparison.

## Category difference

Ad hoc evidence is usually optimised for speed:

- grab the logs
- take screenshots
- copy the payload that seems relevant
- send a note explaining what happened

That can be enough for quick debugging. It is weak when the next step is customer handoff, internal audit, privacy review, or a serious incident retrospective.

ATB is optimised for:

- one portable bundle as the source record
- local verification before review begins
- masked review with explicit reveal logging
- deterministic evidence export when a formal pack is needed

## Practical comparison

| Question | Logs, screenshots, ad hoc exports | ATB |
| --- | --- | --- |
| What is the source record? | Whatever was copied at the time | One local bundle file |
| Can a reviewer verify integrity independently? | No | Yes, with `atb verify` |
| How are sensitive fields handled during review? | Depends on manual redaction | Masked by default with explicit reveal logging |
| What does customer handoff look like? | Notes, screenshots, or continued platform access | Portable bundle plus optional deterministic export |
| Is the review pack repeatable? | Usually no | Yes, by design |

## When ATB is the better fit

Use ATB first when:

- the reviewer is outside the delivery team
- the review outcome needs to be retained or handed off
- privacy-sensitive data makes ad hoc copying risky
- the team needs to prove the record has not been altered

## When ad hoc evidence is fine

Use logs and screenshots when:

- one engineer is debugging an issue in the moment
- there is no need for a portable artefact after the fix
- no external reviewer, customer, or audit function is involved

## Boundary

ATB does not replace every debugging tool. It gives teams a stronger evidence layer when the default alternatives are too fragile to survive review, handoff, or audit.
