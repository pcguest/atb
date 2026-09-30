# W3 Practitioner & Harness Baseline Report

**Programme:** ATB Ecosystem Ground-Truth and Harness Baseline

**Mode:** INTERNAL DOGFOOD / DRY-RUN

**Participant status:** AGENT-EXECUTED — NO HUMAN PARTICIPANT

This report records repository-observed and internally executed evidence. It
does not report observed human comprehension or practitioner usability.

## Repository state

ATB is the public, open-source technical foundation. Mortise and Tenon are
private products and are outside the scope of this public repository report.
ATB must remain usable without either private product.

The W3 harness baseline was merged via PR #26; see git history. The baseline
established repository instructions in `AGENTS.md`, linked them from
`CONTRIBUTING.md`, and retained the internal dogfood record in
`w3-practitioner-dryrun.md`.

The durable validation entry points are:

- `make hygiene-quick`
- `make test-go`
- `make test-golden`
- `go test ./...`

## Harness boundary

The repository harness consists of version-controlled instructions,
documentation, deterministic scripts, tests, and GitHub workflows. Host-level
agent configuration, local machine layout, task plans, and private repository
state are not part of the ATB harness and are not inventoried here.

No ATB-specific agent skill was added during the baseline work. Future skills
should be justified by repeated, repository-observed need and backed by a
deterministic command or test.

## Internal dogfood evidence

The clean-room run exercised installation, source build, SDK installation,
bundle creation, verification, incident review, tamper detection, handoff, and
web checks. Raw command output is retained locally under `.local/runs/` and is
not committed. `w3-practitioner-dryrun.md` maps each reported result to its
local log or marks it `NOT RE-RUN`.

The run established these bounded observations:

- Go installation, source build, and Python and TypeScript SDK installation
  executed successfully in the recorded environment.
- Intact-bundle verification executed with exit code 0.
- Modified, reordered, and removed-record test cases were rejected by the
  verifier.
- A copied bundle verified from a second local path.
- Web accessibility and investigation end-to-end suites completed with all
  recorded specs passing.
- `make demo-incident` failed twice because the Python example searched for
  `tool_without_approval` in Markdown output containing
  `tool\_without\_approval`. Both failed runs produced the same bundle SHA-256.

The last item invalidates the earlier A5 and incident-workflow success claims.
It is a deterministic example-harness defect, not an integrity-verification
failure.

## Product and trust claims

ATB proves the integrity of supplied evidence. It does not prove that capture
was complete, that the recorded claims are true, or that a system is safe or
compliant. Integrity, coverage, corroboration, and custody remain separate
dimensions; no result in this report is a composite trust, confidence, safety,
compliance, or assurance claim.

The public/private dependency direction is an architectural invariant:
private products may depend on ATB, while ATB must not depend on private
Mortise or Tenon implementation or services.

## Empirical limits

The run was performed by an internal agent on a maintainer machine with the
required toolchains installed. It did not observe an unfamiliar practitioner,
measure human interpretation, or establish whether instructions are easy to
follow. Those questions require a real participant study.

## Current disposition

The repository baseline is established. The recorded practitioner path remains
on hold until the deterministic `demo-incident` assertion defect is fixed and
the corrected path is executed, or the founder explicitly defers it.
