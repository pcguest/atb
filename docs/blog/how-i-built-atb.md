# Building ATB: A Solo Developer Project

ATB (Agent Trace Bundle) started from one problem: AI agents are hard to audit after the fact.

When an agent makes a bad decision, most teams only have logs that are hard to trust and harder to replay.
I wanted a format that was simple, local-first, and cryptographically verifiable.

## The Core Bet

I built ATB around three constraints:

1. Local-first by default
2. Integrity verification built in
3. Developer ergonomics over platform complexity

That translated into a newline-delimited JSON bundle format where each event is hash-chained.
If any event changes, verification fails.

## Why Go + Python + TypeScript

- Go for a fast, portable CLI with minimal runtime overhead
- Python because that is where many AI workflows start
- TypeScript for modern app and agent tooling

Cross-language consistency mattered from day one, so I anchored hashing on RFC 8785 canonical JSON.

## What Shipped in v0.9.0-beta

- CLI commands for init, append, snapshot, verify, and local trace viewing
- Python SDK packaging and publish automation
- TypeScript SDK scaffold with typechecking and package-lock reproducibility
- Cross-platform CI on macOS, Linux, and Windows
- Public example bundles in `examples/` to keep the format grounded in real workflows

## Decisions That Increased Velocity

### 1. Minimal interfaces first

I shipped the smallest usable command set and deferred non-essential flags.
That made bugs obvious and testing cheap.

### 2. No framework for the viewer

`atb view` is plain Go + embedded HTML.
No frontend build pipeline, no dependency graph explosion.

### 3. Automation before scale

I added workflows for release notes, issue triage labels, weekly feedback digests, and dependency updates early.
That keeps context-switching low as the project grows.

## What I Got Wrong

- I initially overestimated how quickly external users would discover the project without active distribution.
- I let docs drift from real CLI behavior in early commits.
- I delayed a clean feedback capture loop by a few days.

## What’s Next

The next step is not a broad hosted platform.

I am keeping the local-first core intact and validating a narrower question first: whether teams need secure bundle handoff badly enough to justify a minimal `atb push` path.

If that demand is real, the right shape is encrypted transfer that preserves local control. If it is not, ATB stays focused on local verification, incident review, and portable evidence.
