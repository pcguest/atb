# Tenon family visual grammar

This is the reusable presentation contract for Tenon, ATB, and future Mortise
surfaces. It records the v1.16 ATB implementation; it does not create a Mortise
screen or a cross-product runtime dependency.

## Product emphasis

- **Tenon** owns umbrella identity, onboarding, and family navigation.
- **ATB** is evidence-first: calm, dense forensic investigation with sequence,
  provenance, and bounded conclusions visible before decoration.
- **Mortise** may later emphasise custody, organisation, and retention while
  consuming the same tokens and interaction semantics.

Shared controls use familiar labels, sentence case, restrained borders, and
one strong focus treatment. Status colour always carries text or an icon and
never becomes a composite “health” score.

## Tokens

The source of truth is `web/app/globals.css` and `web/tailwind.config.ts`.

| Purpose              | Token                                           | Rule                                                                             |
| -------------------- | ----------------------------------------------- | -------------------------------------------------------------------------------- |
| Canvas and elevation | `background`, `surface-1..3`, `card`, `popover` | Elevation comes from small tone/border changes, not decorative gradients.        |
| Text                 | `foreground`, `muted-foreground`                | Human label first; IDs and hashes use mono and wrap/truncate deliberately.       |
| Interaction          | `primary`, `ring`, `border`, `muted`            | Blue is the shared action/focus colour; hover remains quieter than selection.    |
| Evidence state       | `verified`, `warning`, `danger`, `unknown`      | Green = verified, amber = attention/inconclusive, red = failure, grey = unknown. |
| Shape                | `radius`                                        | Compact radii; pills are reserved for short status badges.                       |
| Type                 | `font-sans`, `font-mono`                        | Sans for language, mono for evidence identifiers and code.                       |

The base spacing unit is Tailwind's 0.25rem. Controls are normally 2–2.5rem
high; dense rows use 0.75–1rem padding; major surfaces use 1.25–1.75rem. A
single surface header establishes eyebrow, title, explanation, and optional
action.

### Semantic convergence

Existing `background`, `card`, `surface-1..3`, `muted-foreground`, `primary`,
`ring`, and status names remain compatible. New components can use the more
specific semantic names below; raw palette values belong only in globals.css.

| Role           | Tailwind tokens                                                                                                  |
| -------------- | ---------------------------------------------------------------------------------------------------------------- |
| Surfaces       | `surface-page`, `surface-sidebar`, `surface-panel`, `surface-raised`, `surface-selected`, `surface-code`         |
| Text           | `text-primary`, `text-secondary`, `text-tertiary`, `text-technical`, `text-disabled` (e.g. `text-text-tertiary`) |
| Borders        | `border-subtle`, `border-standard`, `border-selected`, `border-status`                                           |
| Interaction    | `interaction-hover`, `interaction-focus`, `interaction-selected`, `interaction-pressed`                          |
| Evidence state | `verified`, `warning`, `failed` (alias of `danger`), `unknown`, `untrusted`                                      |

Blue means interactive or selected. Green means verified, amber means qualified
or untrusted, and red means failed. Event families use neutral technical text;
type labels carry their identity without a rainbow. Status always includes text.

Use 4/8-based spacing, the existing small/medium/large radius set, 12px technical
or secondary text, 14px body/section controls, 20–24px page titles, and 36–60px
display text only on the public homepage. Micro-labels stay at least 11px. Text
sizes do not excuse low contrast: tertiary text is shared with muted foreground
and must pass 4.5:1 on its actual surface. CodeDemo’s former hardcoded #6b7280
install labels now consume this token on `surface-code`; strict axe remains on.

The public product panel uses a real screenshot of the embedded investigation
at `/product/investigation.png`, accompanied by a fixture caption and a link to
the full-size image. Marketing copy and screenshots must be reviewed together;
synthetic records must not be presented as a product capture.

## Investigation composition

ATB View keeps one stable route through Incident → Findings → Timeline →
Context → Relationships → Evidence → Trust. Findings and timeline rows open
the exact supporting event. Relationships are table/list-first; the graph is
an explicitly opened secondary representation. Trust answers three independent
questions: record integrity, selected-profile coverage, and external custody.

The shipped component grammar is:

```text
AppShell
├── verification status
├── Navigation / mobile Tabs
├── PageHeader + CommandPalette + presentation selector
└── SurfaceHeader
    ├── FindingRow / TimelineRow / EvidenceRow / relationship list
    ├── EmptyState / LoadingState / ErrorState
    ├── Inspector + Code/JSON block + Hash/ID + Copy action
    └── optional Tooltip / graph disclosure
```

The command palette is the current modal/dialog primitive. There is no retained
drawer, sheet, generic popover, or text-filter component in the primary flow;
adding an unused abstraction would not improve v1.16.

## State audit

“N/A” means the state has no meaning for that retained component, rather than
an unimplemented visual state.

| Component                            | Default / hover / focus / active                                              | Selected / disabled                                                               | Loading / empty / error                                                                           |
| ------------------------------------ | ----------------------------------------------------------------------------- | --------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| App shell and page header            | Stable hierarchy; sticky header; no decorative interaction                    | Current bundle and surface remain visible                                         | Top-level loading and recoverable local-session error replace the content safely                  |
| Desktop navigation                   | Quiet default, muted hover, shared two-pixel focus ring, immediate activation | Border + accent wash + `aria-current`; disabled N/A                               | N/A                                                                                               |
| Mobile tabs                          | Scrollable default, shared focus ring, immediate activation                   | Accent wash + `aria-selected`; disabled N/A                                       | N/A                                                                                               |
| Surface header                       | Static explanatory hierarchy; action inherits button states                   | N/A                                                                               | N/A                                                                                               |
| Finding row                          | Border strengthens on hover; evidence links have the shared focus ring        | Severity badge; disabled N/A                                                      | Findings surface has derivation loading, bounded empty, and recovery text                         |
| Timeline row                         | Full-row button, muted hover, inset focus ring, immediate activation          | Current selection is expressed after opening Evidence                             | Timeline has loading, bounded empty, and recovery text                                            |
| Relationship list / graph disclosure | Row hover + inset focus; disclosure has keyboard focus                        | Graph open state is native `details`; graph disables on invalid integrity/loading | Empty relationship derivation is explicit; graph loading is disabled rather than stale            |
| Evidence row                         | Full-row hover/focus and immediate activation                                 | Accent border/wash + `aria-pressed`; reveal and copy disable when unavailable     | Pagination has loading feedback; missing requested sequence and no-record cases are distinct      |
| Inspector / JSON                     | Read-only scroll container; masked-field action has hover/focus               | Reveal disables during mutation or invalid integrity                              | Reveal error is visible; no selected event gives an instruction                                   |
| Status badge                         | Text + icon + semantic border/tone                                            | N/A                                                                               | Verified, warning/inconclusive, failed, and unknown/untrusted are isolated fixtures               |
| Button / icon button / copy          | Default border, muted hover, ring focus, pressed feedback from action result  | Reduced opacity and no pointer events when disabled                               | Copy changes to “Copied”; verification and reveal expose pending/error feedback                   |
| Command palette dialog / search      | Trigger and results have hover/focus; arrows move the active option           | Active option uses `aria-selected`; unavailable actions are omitted               | Empty search is explicit; Escape closes and restores trigger focus                                |
| Selector / tooltip                   | Familiar Radix semantics, highlighted item, ring focus                        | Selected item carries a check; disabled N/A                                       | N/A                                                                                               |
| Skeleton / empty / error             | Static and semantically distinct                                              | N/A                                                                               | Skeleton preserves layout; status uses `role=status`; failure uses `role=alert` and a next action |

## Accessibility and motion

Every interactive element needs an accessible name independent of decoration.
Keyboard focus uses the shared `ring` token; dialogs restore focus; native table,
list, definition-list, and disclosure semantics are preferred. Long IDs break
or truncate with access to the exact copy value, and JSON scrolls within a
bounded inspector. The global reduced-motion query collapses animations and
transitions. Firefox, narrow viewport, zoom, contrast, keyboard navigation,
and axe are release gates rather than visual preferences.

## Research boundary

Reactive loading, mutation, interruption, and status patterns may be borrowed
from agent UI research, but ATB does not depend on LangGraph and does not expose
hidden reasoning. Future adapters must map typed application state into these
components. Only explicitly captured summaries, decisions, evidence, tool
states, and provenance may be shown.
