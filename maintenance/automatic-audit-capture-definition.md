# Automatic audit capture definition

Status: architectural definition.
Purpose: define what ATB means by automatic audit capture and how capture completeness should be communicated.

## Core position

ATB is fundamentally an integrity engine.

Automatic audit capture is the layer that attempts to maximise evidence intake while preserving the integrity guarantees of the core bundle model.

ATB must never imply omniscience.

The system can only prove integrity of:

- events it observed;
- events it was explicitly given;
- metadata it recorded;
- artifacts it hashed;
- signatures and timestamps it can verify.

ATB cannot prove:

- provider-side behaviour it never observed;
- hidden prompts not surfaced to the runtime;
- network traffic outside interception scope;
- correctness of the model output;
- intent of the operator;
- completeness of imported historical logs unless explicitly marked.

## Definitions

### Live capture

Events recorded while the workflow is executing.

Examples:

- SDK wrapper events;
- CLI capture session events;
- tool invocation events;
- local process execution events;
- MCP operation events;
- approval workflow events.

### Retrospective import

Historical or external events imported after execution.

Examples:

- imported provider logs;
- imported JSON traces;
- imported chat histories;
- imported CI artifacts.

Retrospective imports must be labelled as retrospective provenance.

### Audit profile

A named definition describing:

- required event classes;
- required metadata fields;
- capture expectations;
- verification expectations;
- known blind spots.

Profiles allow ATB to distinguish:

- tamper-evident but incomplete capture;
- structurally complete capture for a supported workflow.

## Proposed profiles

### cli-basic

For shell or automation workflows.

Required events:

- session_start
- process_start
- command
- stdout_hash
- stderr_hash
- exit_status
- process_end
- session_end

### python-agent

For Python-based agent workflows.

Required events:

- session_start
- model_request
- model_response
- tool_call
- tool_result_hash
- approval_decision
- error
- session_end

### typescript-agent

Equivalent expectations for TypeScript and Node runtimes.

### mcp-session

For MCP-aware execution environments.

Required events:

- server_start
- tool_registration
- tool_invocation
- tool_result_hash
- permission_decision
- session_end

## Verification semantics

`atb verify` should support profile-aware validation.

Examples:

- `atb verify --profile cli-basic`
- `atb verify --profile python-agent`
- `atb verify --profile audit`

Verification output should distinguish:

- integrity success;
- profile completeness success;
- profile completeness warnings;
- unverifiable imported sections;
- unsupported algorithms;
- missing required events.

## Automatic instrumentation goals

Automatic instrumentation should minimise manual developer work.

Desired command:

```bash
atb audit init
```

Desired outcomes:

- create local config;
- select capture profile;
- print framework integration snippet;
- validate write permissions and bundle path;
- emit test event;
- verify the emitted bundle.

## Required adapter direction

### Python

Potential support:

- generic context manager;
- decorator-based capture;
- OpenAI SDK wrapper;
- LangChain adapter;
- LangGraph adapter.

### TypeScript

Potential support:

- generic runtime wrapper;
- OpenAI SDK wrapper;
- LangChain adapter;
- LangGraph adapter.

### CLI

Potential support:

```bash
atb capture run -- npm test
```

Should record:

- process metadata;
- command line;
- command hash;
- start/end timestamps;
- exit status;
- stdout/stderr hashes;
- environment redaction policy.

## Capture transparency

ATB should explicitly report visibility boundaries.

Examples:

- "Provider-side logs not verified"
- "Network traffic not intercepted"
- "Imported history marked retrospective"
- "Tool output recorded by hash only"
- "Sensitive payloads redacted"

The system should prefer transparent partial evidence over overstated certainty.

## Final-state expectation

Automatic audit capture is considered production-ready when:

- supported workflows require minimal manual instrumentation;
- profile completeness checks are enforced;
- blind spots are machine-reported;
- retrospective imports cannot masquerade as live capture;
- evidence can be verified offline;
- SDKs and CLI routes produce structurally compatible bundles;
- local viewer, CLI, and exports all present the same evidence model.
