# UI/UX heuristic audit (v1.15.0 bedrock)

From code and markup alone, before any screenshot round. Two surfaces: the ATB
local viewer (`web/`, the `/view` dashboard) and the Mortise marketing site
(`mortise/site/index.html`). Findings are tagged blocker, major, minor,
or polish, with the Nielsen heuristic and the concrete fix. The
screenshot-dependent items are isolated at the end so one visual round is
decisive.

The only prior design note in the tree is `docs/spec-dashboard.md`; this audit
reconciles against it rather than starting fresh.

## Surface A — ATB local viewer

Rendered composition is `app/view/layout.tsx` (verification banner) wrapping
`app/view/page.tsx` (three-column dashboard). `DashboardShell` is **not imported
anywhere** and does not render.

| # | Sev | Finding | Heuristic | Fix |
| --- | --- | --- | --- | --- |
| A1 | **blocker** | `EventInspector.tsx` tells the auditor that revealing "writes a `privacy.reveal` event **to the bundle on disk**. This action is permanent" (tooltip, lines 127-129) and "writes privacy.reveal event **to bundle**" (line 133). The v1.15.0 fix routes reveals to a separate `.reveals` sidecar; the authoritative bundle is byte-unchanged (proven: sha256 identical before/after). The headline trust copy states the opposite of what the code does. | Match between system and the real world; error prevention | Rewrite both strings: reveal writes to the independent `.reveals` sidecar with its own hash chain; the authoritative bundle is never modified. Code-only, no screenshot needed. |
| A2 | major | Dead component `DashboardShell.tsx`: contains a second `<h1>`, a second `<main id="dashboard-content">`, a non-functional role nav (buttons with no handler), and the grandiose "Enterprise Trust Control Plane" string. It never renders, but it is a landmine: a future wiring of it would duplicate landmarks/ids and ship dead controls and marketing voice. | Consistency and standards; aesthetic and minimalist design | Delete the file (and its unused nav config), or, if kept, gut it to the real layout. Removes the duplicate-id/landmark risk and the marketing phrasing in one stroke. |
| A3 | major | Tamper banner (`VerificationBanner.tsx`, invalid branch) shows only "⚠ TAMPER DETECTED" with no locus: no failing seq, no chain length, no bundle path. The CLI gives "tamper detected at event 5 (seq 5)". A sceptical auditor sees a red bar with no diagnosis. | Help users recognise, diagnose, and recover from errors; visibility of system status | Surface the failing seq (and ideally expected/got, or at least "run `atb verify <path>` for the failing record") in the invalid banner; keep the bundle path visible on FAIL as it is on PASS. |
| A4 | minor | `EventInspector.tsx` uses hardcoded `slate-*` colours (`bg-slate-950`, `border-slate-800`, `text-slate-200`) instead of the design tokens (`bg-card`, `border-border`, `text-foreground`) the rest of the dashboard uses. A `ThemeToggle` exists, so in light theme the inspector stays dark. | Consistency and standards | Replace slate utilities with the semantic tokens so the inspector tracks the theme. |
| A5 | minor | Duplicate "Inspector" heading: the right rail in `page.tsx` (lines 211-213) renders an "Inspector" header, and `EventInspector.tsx` (lines 80-82) renders its own. | Consistency and standards; minimalist design | Drop one of the two headers. |
| A6 | minor | `page.tsx` three-column layout is fixed-width (`w-64` + `flex-1` + `w-[420px]`) with `overflow-hidden` and no responsive breakpoints. The responsive sidebar logic lives only in the dead `DashboardShell`. On a narrow window the columns are cramped or clipped. | Flexibility and efficiency of use | Add a breakpoint that collapses or stacks the rails below ~`md`. Lower priority for a localhost desktop tool. |
| A7 | minor | For auditor and executive roles, raw events do not load (`canViewRawData` is engineer-only), so the 340px DAG section and the 420px inspector rail render empty ("Select an event to inspect details."). The space reads as broken rather than intentionally restricted. | Aesthetic and minimalist design; match to real world | Hide or replace the graph/inspector rails for non-engineer roles with a short note ("Raw trace is available in the Engineer role"). |
| A8 | polish | `StatsOverview` shows the verification value raw and lowercase (`verification?.status` -> "valid"/"invalid"). | Recognition rather than recall | Capitalise and pair with the same lock/shield icon the banner uses, for one consistent status vocabulary. |
| A9 | polish | "Viewer Health N/100" (auditor and executive panels) is a composite of verification + recency. It sits next to Integrity and CAS and risks being read as a third integrity verdict. | Match between system and the real world | Add a one-line tooltip stating what Viewer Health measures and that it is not the integrity result. |

Strengths worth keeping: the banner has correct `role="status"`/`role="alert"`
with `aria-live="assertive"` and a `data-tamper` document attribute; the head
hash is shown truncated with a full-value `title`; `RoleSelector` is a proper
labelled Radix select with focus rings; the reveal flow already carries an
honest "tamper-evident" caption (it just names the wrong file — see A1).

## Surface B — Mortise marketing site

`site/index.html`, self-contained, no external assets (verified), `lang="en-GB"`.

| # | Sev | Finding | Heuristic | Fix |
| --- | --- | --- | --- | --- |
| B1 | major | Hero CTA reads "Talk to **us** about a design partnership" (line 122). House rules forbid first-person-plural marketing voice. The rest of the copy is correctly third person. | Consistency and standards (project voice); match to real world | Reword to "Discuss a design partnership" or "Start a design-partnership conversation". |
| B2 | minor | No `<main>` landmark; all content sits in `<section>`s inside `.wrap` divs. No skip-to-content link. (Already noted in `ACCEPTANCE.md`.) | Accessibility and standards | Wrap the section stack in `<main id="main">` and add a visually-hidden skip link in the header. |
| B3 | polish | The verification ticks are bare `&#10003;` glyphs with no text alternative; meaning is carried only by the adjacent cell text. | Accessibility | Add `aria-hidden="true"` to the glyph (meaning is in the adjacent text) or a visually-hidden "verified" label. |
| B4 | polish | Table headers lack `scope="col"`. | Accessibility and standards | Add `scope="col"` to the two `<th>`. |

Strengths worth keeping: single restrained accent, honest-limits panel intact,
the "what a third party can verify without trusting Mortise" table is the
strongest trust artefact on the page, the Article 12 line is the approved honest
wording, and the open-core boundary matches the ATB README.

## Items that need the single screenshot round to settle

These cannot be judged from code; target the one visual round at exactly these:

1. **Banner colour contrast** — green-400 on green-950 (PASS) and red-300 on
   red-950 (FAIL) against WCAG AA. Code shows the palette; only a render
   measures contrast.
2. **Empty-rail appearance for auditor/executive** (A7) — whether the empty
   340px graph and 420px inspector look intentional or broken.
3. **Narrow-window layout** (A6) — does the fixed three-column grid clip or
   merely crowd.
4. **TraceGraph legibility** — dagre top-down layout density and label overlap
   on a real bundle.
5. **EventInspector in light theme** (A4) — confirm the slate hardcoding looks
   wrong, and that the token fix reads correctly in both themes.
6. **Mortise site rendered hierarchy** — hero pill/headline/lead spacing and
   whether the two-column open-core boundary reads clearly at desktop and at the
   720px breakpoint.

Everything else above (A1-A3, A5, A8-A9, B1-B4) is settled from the markup and
can be fixed without a screenshot.
