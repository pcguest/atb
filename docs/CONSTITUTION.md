# ATB Project Constitution

## 1. Mission
Cryptographically verifiable audit trails for AI agent workflows. AI-Readable, Human-Trustable.

## 2. Non-Negotiables (The Five Pillars)
1. **Zero-Knowledge by Default:** Keys/passwords never leave client.
2. **Local-First, Cloud-Optional:** Offline works; cloud enhances.
3. **Integrity-Verified Everywhere:** Hash chain on every load/save.
4. **Observable but Private:** Metrics exist; user data never exposed.
5. **Documented or It Doesn't Exist:** If not in docs/, it's not ready.

## 3. Decision Framework (The Paddy Test)
Before any change, ask:
1. "Would I use this tomorrow to debug my own agent workflow?"
2. "Does this make ATB more verifiable, or just more complex?"
3. "Can an AI parse the output without heuristics?"
4. "Can a human read the output and feel confident in the integrity claim?"

If any answer is "no", reconsider the change.

## 4. Architecture Principles
- **Minimal Surface Area:** Prefer CLI flags over config files.
- **Deterministic Output:** Same input → same output (verified via golden tests).
- **Cross-Language Parity:** Go/Python/TS must produce identical canonical JSON/hashes.
- **Fail Fast:** Verification errors block exports/archives immediately.
- **Tamper-Evident:** All logs (bundles, archives) are hash-chained.

## 5. Development Workflow
- **One Task at a Time:** Complete fully before moving to next.
- **Verify Locally:** Tests must pass before commit.
- **Dogfood Every Change:** Use ATB to track ATB development.
- **No Hypothetical Scale:** Build for today's needs, not tomorrow's hypotheticals.

## 6. Agent Coordination Rules
- **Handoff Required:** When switching agents, generate a State of the Union document.
- **No Overlap:** Agents own disjoint file sets unless coordinating.
- **Fresh Chat Per Phase:** Prevents context drift and hallucinations.

## 7. Success Metrics
- **CI Green:** 100% pass rate on all workflows.
- **Parity Verified:** Go/Python/TS byte-identical outputs.
- **Docs Aligned:** All claims match implementation.
- **Solo Sustainable:** <4 workflows, <1 hour/week maintenance.
