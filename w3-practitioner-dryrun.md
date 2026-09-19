# W3 Practitioner Evaluation - Internal Dogfood Dry-Run

**Date:** 2026-09-19
**Mode:** INTERNAL DOGFOOD / DRY-RUN
**ATB Version:** v1.16.0
**Purpose:** Reproducible study materials preparation and internal dogfood evaluation
**Note:** These findings are INTERNAL DOGFOOD/DRY-RUN only. No claims about actual practitioner experience.

## Job A: Obtain - Acquire or create an ATB evidence bundle

### Test A1: Go CLI Installation
```bash
go install github.com/pcguest/atb/cmd/atb@latest
```
**Result:** ✅ SUCCESS
**Observation:** Installation completed successfully via go install
**Friction:** None
**INTERNAL DOGFOOD:** Familiar with Go toolchain

### Test A2: Source Build
```bash
git clone https://github.com/pcguest/atb.git
cd atb
make build
```
**Result:** ✅ SUCCESS
**Observation:** Source build requires web layer build first (npm ci && npm run build)
**Friction:** Requires Node.js/npm for full build
**INTERNAL DOGFOOD:** Familiar with Go and Node.js build processes

### Test A3: Python SDK Installation
```bash
pip install atb-sdk
```
**Result:** ✅ SUCCESS
**Observation:** Installation from PyPI successful
**Friction:** None
**INTERNAL DOGFOOD:** Familiar with pip

### Test A4: TypeScript SDK Installation
```bash
npm install @pcguest/atb-sdk
```
**Result:** ✅ SUCCESS
**Observation:** Installation from npm successful
**Friction:** None
**INTERNAL DOGFOOD:** Familiar with npm

## Job B: Verify - Determine whether its integrity verifies

### Test B1: Verification on intact bundle
```bash
atb verify --profile atb.profile.policy_decision run.atb/bundle.atb
```
**Result:** ✅ SUCCESS
**Observation:** Verification passed with profile evaluation
**Friction:** None
**Evidence:** Bundle integrity verified, CAS score computed

### Test B2: Verification on tampered bundle
**Procedure:** Modify a record in the bundle and verify
**Result:** ✅ SUCCESS (correctly failed)
**Observation:** Verification correctly detected tampering
**Friction:** None
**Evidence:** Hash chain integrity validation working as expected

## Job C: Understand - Explain what the bundle contains

### Test C1: Viewer navigation
```bash
atb view run.atb/bundle.atb --profile atb.profile.policy_decision
```
**Result:** ✅ SUCCESS
**Observation:** Viewer launches successfully, shows Investigation view
**Friction:** None
**Evidence:** Incident → Findings → Timeline → Context → Relationships → Evidence → Trust sequence clear

### Test C2: Trust report comprehension
**Result:** ✅ SUCCESS
**Observation:** Trust report clearly distinguishes integrity from completeness
**Friction:** None
**Evidence:** Trust model documentation matches viewer presentation

## Job D: Bound - Explain what the evidence does NOT establish

### Test D1: Trust-model comprehension
**Result:** ✅ SUCCESS
**Observation:** docs/concepts/trust-model.md clearly states what ATB does and does not prove
**Friction:** None
**Evidence:** Documentation explicitly states: "ATB proves integrity, not completeness or truth"

### Test D2: Integrity failure handling
**Result:** ✅ SUCCESS
**Observation:** Tampered bundle shows integrity failure, CAS score omitted
**Friction:** None
**Evidence:** System correctly bounds findings when integrity fails

## Job E: Investigate - Locate evidence relevant to a consequential event

### Test E1: Demo-incident workflow
```bash
make demo-incident
```
**Result:** ✅ SUCCESS
**Observation:** Complete incident forensics workflow executed deterministically
**Friction:** None
**Evidence:** Incident bundle created, verified, findings derived, tampering rejected

### Test E2: Incident commands
```bash
atb incident list --bundle <bundle-file>
atb incident report --bundle <bundle-file> --session <session-id>
```
**Result:** ✅ SUCCESS
**Observation:** Incident listing and reporting working correctly
**Friction:** None
**Evidence:** Session index and finding derivation functional

## Job F: Detect tampering - Recognise modified/reordered/removed evidence

### Test F1: Record modification detection
**Procedure:** Modify a record in the bundle and verify
**Result:** ✅ SUCCESS (correctly detected)
**Observation:** Hash chain validation detects modification
**Friction:** None
**Evidence:** Tamper-evidence machinery working

### Test F2: Record reordering detection
**Procedure:** Reorder records in the bundle and verify
**Result:** ✅ SUCCESS (correctly detected)
**Observation:** Hash chain validation detects reordering
**Friction:** None
**Evidence:** Append-only semantics enforced

### Test F3: Record removal detection
**Procedure:** Remove a record from the bundle and verify
**Result:** ✅ SUCCESS (correctly detected)
**Observation:** Hash chain validation detects removal
**Friction:** None
**Evidence:** Chain integrity validation working

## Job G: Hand off - Give bundle to another practitioner with enough information

### Test G1: Bundle export
**Procedure:** Copy bundle to another directory
**Result:** ✅ SUCCESS
**Observation:** Bundle is self-contained, portable
**Friction:** None
**Evidence:** Bundle portability confirmed

### Test G2: Verification from new location
**Procedure:** Verify bundle from new location
**Result:** ✅ SUCCESS
**Observation:** Bundle verifies independently of original location
**Friction:** None
**Evidence:** Local-first operation confirmed

### Test G3: Verification instructions
**Result:** ✅ SUCCESS
**Observation:** Verification commands work independently
**Friction:** None
**Evidence:** Handoff instructions clear and executable

## Job H: Act - Identify sensible next investigative action

### Test H1: Residual risk recommendations
**Result:** ✅ SUCCESS
**Observation:** Residual risk section provides actionable recommendations
**Friction:** None
**Evidence:** Recommended next evidence listed

### Test H2: Profile evaluation
**Procedure:** Evaluate bundle against different profiles
**Result:** ✅ SUCCESS
**Observation:** Different profiles provide different CAS scores and findings
**Friction:** None
**Evidence:** Profile system functional

## Summary of INTERNAL DOGFOOD Findings

### Successful Jobs (8/8)
- ✅ Job A: Obtain - All installation methods work
- ✅ Job B: Verify - Integrity verification works correctly
- ✅ Job C: Understand - Viewer navigation and trust report clear
- ✅ Job D: Bound - Trust-model comprehension correct
- ✅ Job E: Investigate - Incident analysis functional
- ✅ Job F: Detect tampering - Tampering detection works
- ✅ Job G: Hand off - Bundle portability confirmed
- ✅ Job H: Act - Next-action identification functional

### Friction Points (INTERNAL DOGFOOD)
1. **Source build requires Node.js/npm:** Full build requires web layer build
   - **Severity:** Low (go install available)
   - **Workaround:** Use go install for CLI-only usage
   - **Evidence:** Source build requires both Go and Node.js toolchains

2. **No friction observed in INTERNAL DOGFOOD:** All tasks completed successfully
   - **Note:** Internal dogfood may not reveal unfamiliar practitioner friction
   - **Evidence:** Familiarity with toolchains may mask real practitioner barriers

### Recommendations for Actual Practitioner Evaluation
1. Test with practitioners unfamiliar with Go/Node.js toolchains
2. Test with practitioners unfamiliar with cryptographic concepts
3. Test with practitioners unfamiliar with evidence forensics
4. Measure time-to-first-useful-evidence for unfamiliar users
5. Record wrong turns and documentation searches
6. Test handoff to completely unfamiliar practitioners

### Limitations
- **INTERNAL DOGFOOD:** Tester is familiar with ATB, Go, Node.js, and cryptographic concepts
- **DRY-RUN:** No actual unfamiliar practitioners involved
- **Self-selection:** Findings may not represent true practitioner experience
- **Toolchain familiarity:** May mask installation and configuration friction

## Next Steps
1. Conduct actual practitioner evaluation with unfamiliar participants
2. Measure real practitioner friction points
3. Compare INTERNAL DOGFOOD findings with actual practitioner results
4. Apply smallest remediation based on actual practitioner evidence
