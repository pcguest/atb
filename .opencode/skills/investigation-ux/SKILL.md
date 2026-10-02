---
name: investigation-ux
description: Use when changing the ATB viewer investigation experience — Run/Findings/Timeline/Context/Relationships/Evidence/Evidence status, progressive disclosure, digests, raw JSON, source-change comparison. Trigger on: web/app/view/**, Cypress investigation specs, viewer API projection, evidence detail, finding detail, acquisition provenance in the viewer.
---

# Investigation UX

## Purpose

Keep the viewer a developer/security/evidence tool: dense, honest, scannable.
Not an AI chatbot SaaS. No dashboards, health scores, trust scores, or
hallucination scores.

## Information architecture (preserve)

Run → Findings → Timeline → Context → Relationships → Evidence → Evidence status

Do NOT add an "Acquisition" top-level page. Acquisition provenance belongs in
progressive disclosure on Evidence detail, Finding detail, and Evidence status.

## When to invoke

- Editing `web/app/view/**` or viewer components
- Editing the investigation projection in `pkg/api/v1/investigation*.go`
- Editing Cypress investigation/a11y specs
- Adding any evidence metadata, digest, raw JSON, or copy affordance

## Bounded procedure

1. Extend an existing surface; do not create a new top-level page.
2. Use progressive disclosure: summary first, expandable metadata on demand.
3. Long digests: truncate visually, keep full value copyable; never lie about length.
4. Source-change: show previous vs current observation and their acquisition times.
5. Keyboard disclosure + deep links; strict axe accessibility must pass.
6. Virtualise only if measurement shows it is needed.
7. Prefer existing primitives/components/tokens; no new design system.

## Stop conditions

- Stop if the change implies completeness, truth, or a universal score.
- Stop if it introduces a dashboard/score/chat aesthetic.
- Stop if it copies proprietary branding or assets.

## Canonical references

- docs/specification/viewer.md
- docs/maintainers/visual-system.md
- docs/research/PROFESSIONAL_UI_GRAMMAR.md
- docs/research/MARKET_UI_PATTERN_MATRIX.md
- docs/investigate/*.md

## Tests / checks

- web lint, typecheck, unit tests, production build
- Cypress investigation journey (Firefox), strict a11y
- keyboard navigation; desktop / narrow / small viewport
- long digest; legacy bundle; mixed acquisition bundle; SOURCE_RECORD_CHANGED

## Prohibited interpretations

- A rendered relationship is not a causal claim.
- "Evidence status" is not a trust or safety verdict.
