# Research note — EU AI Act evidence-to-obligation mapping

Status: **regulatory research / scoping note. Not legal advice, not a compliance
claim.** Its purpose is a defensible, honest mapping of *what ATB/Custos evidence
supports* against *what specific Articles actually require*, so the product can
help discharge concrete technical sub-obligations without ever claiming to make a
system "AI-Act compliant." This is the credibility backbone of the regulatory
wedge and the single easiest thing to get wrong by overclaiming.

Sources at the end. Article readings are summarised from the public consolidated
text; treat the official Regulation (EU) 2024/1689 as authoritative.

## 0. The honesty frame (non-negotiable)

The provability ladder already states the limit: ATB proves **integrity** and
documents **coverage**; it does **not** provide "legal or compliance certification
(CAS is not an audit opinion)" (`docs/provability-ladder.md`). Everything below
inherits that. ATB/Custos are **technical measures a provider or deployer can use
to satisfy specific recording, tamper-evidence, retention, and oversight-capture
sub-obligations.** They do not:

- make an AI system high-risk-conformant (that is a system-level, organisational,
  and process undertaking spanning Articles 8–27);
- discharge the *process* obligations (risk management Art 9, data governance
  Art 10, QMS Art 17);
- assert any fact about model quality, human comprehension, or legal conformity.

A mapping that respects this is an asset; one that blurs "we record the logs
Article 12 needs" into "we make you Article 12 compliant" destroys the trust the
whole project is built on.

## 1. Article-by-article mapping

Legend for **ATB support**: ●= directly evidences · ◐= partially · ○= out of scope.

### Article 12 — Record-keeping (provider, design-time) ●

**Requires:** high-risk systems "technically allow for the automatic recording of
events (logs) over the lifetime of the system." 12(2): logs must cover (a) risk /
substantial-modification situations, (b) post-market monitoring, (c) operation
monitoring by deployers. 12(3) (for Annex III 1(a) biometric): at minimum record
the period of each use (start/end date-time), the reference database, the input
data that produced a match, and the natural persons verifying results per 14(5).

**What ATB evidences:** ATB *is* an automatic, tamper-evident event recorder.
Session open/close events carry start/end times; action/tool/policy/override
events plus `ai.action.error` cover operational and risk situations; the capture
scope attestation (`atb.capture.scope`) states the recorder's boundary. For 12(3)
specifically: start/end-of-use → session timestamps; the "natural persons
verifying" → `human_override`/`policy_decision` principal fields (see Art 14).

**Residual (always document):** Article 12 is an obligation on the *provider's
system*; ATB is a component a provider integrates, not the system. ATB cannot
guarantee the system records events it never emitted (the L5 capture-boundary
limit). The reference-database / input-match minima of 12(3) are
domain-specific fields the integrating system must populate.

### Article 14 — Human oversight ◐

**Requires:** systems designed so oversight persons can understand capacities and
limitations, monitor operation, **correctly interpret output**, decide **not to
use / disregard / override / reverse** output, and **intervene or stop**. 14(5):
for biometric ID (Annex III 1(a)), no action on an identification unless
**separately verified and confirmed by at least two natural persons** with
competence and authority.

**What ATB evidences:** the *act* of oversight — `human_override`,
`policy_decision`, and `ai.action.precommit` carry an optional `principal`
(human/agent/tool + `on_behalf_of`) and the override/approval outcome and reason.
ATB can show *that* a human overrode/approved, *when*, and *under what asserted
identity and scope*, and (with sequencing) that the human action preceded the
gated action.

**Residual / the real gap:** ATB records **caller-asserted** identity. It does not
today bind the overseer to a **verifiable external identity** (IdP/OIDC, signed
assertion) — this is the ladder's explicit L3 residual ("caller-asserted identity
without external IdP"). And ATB can never evidence the overseer's *comprehension*
or *competence* (an irreducible limit). For 14(5)'s "two natural persons", ATB can
record two distinct principal assertions but cannot, alone, prove they are two
real, competent, authorised humans. → This is precisely what the roadmap's
**"reviewer identity anchoring (Article 14 gap closure)"** item should close, and
the honest target is *"binds the override to a verifiable identity assertion,"* not
*"proves competent human oversight."*

### Article 17 — Quality management system (provider) ○

**Requires:** a documented QMS (strategies, procedures, techniques) for
conformity. **Process obligation.** ATB is not a QMS. At most, ATB/Custos can be a
*documented technical measure within* a provider's QMS (e.g., "tamper-evident
execution logging is implemented via ATB"). Claim nothing more.

### Article 18 — Documentation keeping ◐

**Requires:** keep the technical documentation (Annex IV), QMS docs, approvals, and
decisions of notified bodies **for 10 years** after the system is placed on the
market, at the disposal of authorities.

**What ATB/Custos support:** the *artifacts* are retainable — bundles, incident
evidence packs, and `verify.report.v1` are self-contained and independently
re-verifiable, which is exactly what "at the disposal of authorities" wants.
Custos custody provides the durable, tamper-evident hold.

**Residual:** the **10-year lifecycle management** (retention scheduling, legal
hold, disposal) is custody/retention infrastructure, much of which lives outside
the ATB runtime per `AGENTS.md`. ATB evidences integrity of what is kept; it does
not by itself manage the retention calendar.

### Article 19 — Automatically generated logs (provider retention) ●

**Requires:** providers "shall keep the logs referred to in Article 12(1),
automatically generated by their high-risk AI systems… to the extent such logs are
under their control… for a period appropriate… of **at least six months**," subject
to data-protection law.

**What ATB/Custos evidence:** ATB bundles **are** such logs, and they are
tamper-evident — strictly stronger than the Article's ask (it requires keeping
logs, not proving they are unaltered). ATB already ships an "EU AI Act retention
guard" (see roadmap current-state) and Custos custody provides the durable hold +
independent re-verification over the retention window. The transparency log
(see [transparency-log.md](./transparency-log.md)) further makes "we kept it,
unaltered, since time T" externally verifiable.

**Residual:** "under their control" and the data-protection interaction (e.g.,
GDPR erasure vs. retention) are deployment-policy questions; ATB's privacy-by-default
capture (body digests, secret-header stripping) helps but does not resolve them.

### Article 20 — Corrective actions and duty to inform ◐

**Requires:** where a system is not in conformity, take corrective action / withdraw
/ recall, and inform distributors, authorities, etc.; relatedly, serious-incident
reporting (Art 73).

**What ATB supports:** the **evidentiary basis** for an incident — `atb incident
report` / `export` produces a session-scoped, independently verifiable forensic
package (integrity, signature provenance, explained findings, timeline). That is
the substantiation an Article 20 corrective action / Article 73 notification rests
on. ATB does not perform the notification or judge conformity; it makes the
incident *defensibly reconstructable*.

### Article 26(6) — Deployer log retention ●

**Requires:** deployers "shall keep the logs automatically generated by that
high-risk AI system, to the extent such logs are under their control, for a period
appropriate… of **at least six months**," subject to Union/national/data-protection
law.

**What ATB/Custos evidence:** ATB is local-first, so the *deployer* can record and
hold logs "under their control" with no third-party dependency — a strong fit for
26(6). Custos gives the deployer a neutral, tamper-evident custody option for the
six-month (or longer) window, with independent re-verification.

**Residual:** same control/data-protection caveats as Art 19.

## 2. Summary table

| Article | Obligation (short) | Who | ATB/Custos support | Residual to always state |
|---|---|---|---|---|
| 12 | system technically records events over lifetime | provider | ● tamper-evident event recorder, session timing, capture scope | ATB is a component, not the system; 12(3) domain fields must be populated |
| 14 | enable interpret / override / stop; 14(5) two-person | provider | ◐ records the *act* of override/approval + asserted principal | caller-asserted identity (L3 gap); never proves comprehension/competence |
| 17 | quality management system | provider | ○ at most a documented measure *within* a QMS | not a QMS |
| 18 | keep technical docs 10 years | provider | ◐ retainable, self-verifying artifacts | 10-yr lifecycle mgmt is retention infra |
| 19 | keep Art 12 logs ≥6 months | provider | ● bundles *are* tamper-evident logs; custody hold | "under control" + data-protection policy |
| 20 | corrective action + inform | provider | ◐ provides the incident evidentiary basis | does not notify or judge conformity |
| 26(6) | deployer keeps logs ≥6 months | deployer | ● local-first control; neutral custody option | "under control" + data-protection policy |

## 3. What this mapping justifies building (honest, code-shaped)

The mapping turns three roadmap items from vague into concrete and bounded:

1. **Reviewer identity anchoring (Art 14, L3 closure).** Bind a `human_override` /
   `policy_decision` principal to a *verifiable* identity assertion — e.g. accept a
   signed identity token (OIDC ID-token / JWT / detached signature) and record its
   verified subject + issuer + a signature over the decision, so the evidence shows
   "this decision was authorised under an identity an external IdP vouched for,"
   not merely a string. Honest claim ceiling: *verifiable identity assertion*, not
   *proven competent oversight*.
2. **Retention enforcement access logging (Art 12(2)(b)/19/26(6)).** Record an
   audit event whenever retained evidence is accessed or exported, so the retention
   story itself is evidenced. Small, deterministic, in-scope.
3. **Article-keyed compliance evidence-pack export (Art 17–20, 26(6)).** Extend
   `atb incident export` with an optional, **honestly-labelled** index mapping each
   artifact to the sub-obligation it supports *and the residual it does not* — i.e.
   this table, rendered per-pack. The label must say "supports" not "satisfies".

Each is deterministic, evidence-first, and refuses the overclaim. None of them
asserts compliance; they make the *technical* sub-obligations *demonstrable*.

## Sources

- [Article 12 — Record-keeping](https://artificialintelligenceact.eu/article/12/)
- [Article 14 — Human oversight](https://artificialintelligenceact.eu/article/14/)
- [Article 18 — Documentation keeping](https://ai-act-service-desk.ec.europa.eu/en/ai-act/article-18)
- [Article 19 — Automatically generated logs](https://artificialintelligenceact.eu/article/19/)
- [Article 20 — Corrective actions and duty of information](https://ai-act-service-desk.ec.europa.eu/en/ai-act/article-20)
- [Article 26 — Obligations of deployers of high-risk AI systems](https://artificialintelligenceact.eu/article/26/) (see 26(6))
- [Annex IV — Technical documentation](https://ai-act-service-desk.ec.europa.eu/en/ai-act/annex-4)
- Practitioner readings: [FireTail — Article 12 and the logging mandate](https://www.firetail.ai/blog/article-12-and-the-logging-mandate-what-the-eu-ai-act-actually-requires); [Help Net Security — what the AI Act requires for AI agent logging](https://www.helpnetsecurity.com/2026/04/16/eu-ai-act-logging-requirements/); [PipeLab — Article 26 deployer obligations & 6-month retention](https://pipelab.org/learn/eu-ai-act-compliance/)
