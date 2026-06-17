# Launch collateral: v1.15.0 and the Custos front

Ready-to-use text for the launch. Nothing here has been posted, pushed, or
published. House rules apply throughout: British English, restrained, no
certification language. Adjust the security mailbox and the release date before
use. The CHANGELOG currently dates v1.15.0 as 2026-06-15; set it to the actual
tag date.

---

## 1. GitHub release body (tag v1.15.0, repo pcguest/atb)

> ATB v1.15.0
>
> This release makes inspecting evidence safe, makes the Custos custody path
> conform to the ingest contract, and adds an offline compliance pack export. It
> records reviewer identity as caller-provided evidence without claiming to
> verify it.
>
> ### Highlights
>
> - Revealing a masked field in `atb view` no longer writes to the authoritative
>   bundle. Reveal audit events go to a separate `<bundle>.reveals` sidecar with
>   its own hash chain, bound to the source bundle by id and chain head.
>   Inspecting evidence no longer changes it. The sidecar verifies independently.
> - The Custos push path now conforms to the ingest API. `atb intercept --custos`
>   and `atb incident export --custos-endpoint` post the completed bundle to
>   `POST /ingest` and surface the signed receipt.
> - New deterministic offline `atb compliance pack` export: the authoritative
>   bundle, `verify.report.v1`, trust reports, CAS and obligation results,
>   incident artefacts, reference mappings, checksums, and relevant retention
>   operations.
> - Optional digest-only reviewer identity evidence across Go, Python, and
>   TypeScript, labelled caller-provided and not independently verified by ATB.
> - Retention audit events recorded in the separate `.atb/operations.atb` chain.
>
> ### Contract
>
> The `verify.report.v1` consumer contract is unchanged. `report_version` stays
> `verify.report.v1`; the JSON Schema file revision advanced to `.schema.2` for
> the additive optional `reviewer_identities` field. Every existing field keeps
> its name and meaning. See VERSIONING.md.
>
> ### Honest limits
>
> ATB proves the integrity of what was recorded. It does not prove universal
> capture, model correctness, actor identity, or legal compliance by itself, and
> it never certifies compliance.
>
> SDKs: `atb-sdk` on PyPI and `@pcguest/atb-sdk` on npm, both 1.15.0.
> Full notes in CHANGELOG.md.

---

## 2. Pull request description: ATB (release/v1.15.0 into main)

> ### v1.15.0
>
> Eleven signed commits, grouped by subsystem. Per-commit walkthrough in
> RELEASE-REVIEW-v1.15.0.md; re-proved claims in ACCEPTANCE.md.
>
> What changed and why it matters to a reviewer:
>
> - **Reveal no longer mutates evidence.** The load-bearing trust fix. A reveal
>   in the viewer wrote `privacy.reveal` into the authoritative bundle and
>   re-saved it, so inspecting evidence changed its bytes. Reveals now write to a
>   `<bundle>.reveals` sidecar with its own chain. Proven byte-unchanged.
> - **Custos push conforms.** The old client posted to a non-existent `/bundle`
>   endpoint. It now posts the whole bundle to `POST /ingest` and returns the
>   signed receipt. Verified live against a running ingest daemon.
> - **Offline `atb compliance pack`.** Deterministic evidence export, local only.
> - **Additive schema.** Optional `reviewer_identities`, strict-additive against
>   v1.14.5, hash-frozen, `report_version` unchanged. Decision recorded in
>   ACCEPTANCE.md.
> - **Build fix.** Restores `web/out/placeholder.txt` so a clean clone builds
>   (the embed pattern matched nothing without it).
> - **Disclosure contact parameterised** to a single token; personal address
>   remains only in package author metadata by design.
>
> Gates green: `make test-golden` (Go, Python, TypeScript), full `go test ./...`
> (ATB and the custos module), `make hygiene-quick`, version markers all 1.15.0.
> Some files span two concerns; the honest list is in the review document.

---

## 3. Pull request description: Custos (site/marketing-front into main)

> ### Public marketing front
>
> A self-contained static `site/index.html` and `site/README.md`. No build step,
> no JavaScript, no external assets, `lang="en-GB"`. The open-core boundary table
> matches the ATB README content exactly (seven free, seven Custos). The contact
> path is a design-partner mailto, not a pricing table. Honest limits stated:
> Custos produces evidence, it does not issue certificates, and equivocation
> resistance is bounded by the witnesses actually deployed.
>
> Preview: `python3 -m http.server --directory site 8000`. Audited: no external
> assets, all anchors resolve, no em dashes, no ampersand glyphs.

---

## 4. Launch announcement (short plain-text post)

> ATB v1.15.0 is out. ATB is an open, MIT-licensed evidence core for AI agent
> activity: tamper-evident bundles, a SHA-256 hash chain over canonical JSON, and
> offline verification with one CLI.
>
> This release fixes the one thing that would stop a careful reviewer adopting
> it: inspecting evidence no longer changes it. Revealing a masked field now
> writes to a separate sidecar with its own chain, never the authoritative
> bundle. The custody path to Custos conforms to the ingest contract and returns
> a signed receipt. A new offline compliance pack exports the bundle, the
> machine-readable verify report, and the reference mappings in one deterministic
> archive.
>
> ATB proves the integrity of what was recorded. It does not prove universal
> capture, model correctness, or legal compliance, and it never certifies
> anything. The Article 12 evidence mapping states each obligation and what ATB
> does not prove.
>
> Code: github.com/pcguest/atb. SDKs: atb-sdk (PyPI), @pcguest/atb-sdk (npm).

---

## 5. Launch announcement (three-line summary)

> ATB v1.15.0: an open evidence core for AI agent activity. Tamper-evident
> bundles, offline verification, honest Article 12 mapping.
> Inspecting evidence no longer mutates it; reveals go to a verifiable sidecar.
> Open and MIT. Custos adds custody, WORM, and a transparency log on top.
