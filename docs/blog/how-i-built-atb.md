# How I Built ATB in 3 Weeks as a Solo Founder

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

## What Shipped in v1.0.x

- CLI commands for init, append, snapshot, verify, and local trace viewing
- Python SDK packaging and publish automation
- TypeScript SDK scaffold with typechecking and package-lock reproducibility
- Cross-platform CI on macOS, Linux, and Windows
- Dogfooding traces in `dev-log/` to keep myself honest

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

v1.1 focuses on optional cloud sharing (`atb push`) without violating the local-first core.
The working spec uses client-side encryption and time-limited links.

I am explicitly validating demand before implementing this in full.

## Advice for Other Solo Founders

- Ship narrow, then automate repeatable toil.
- Keep your architecture reversible.
- Record your own development process as product telemetry.
- Trust user pull more than roadmap intuition.

If you are building agent systems and need auditability, ATB is open source:

<https://github.com/pcguest/atb>
