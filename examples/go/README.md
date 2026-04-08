# Go End-to-End CLI Demo

This walkthrough creates and verifies a local ATB bundle end to end using the Go CLI.

## Prerequisites

- Go 1.25.0+
- `atb` installed (`go install github.com/pcguest/atb/cmd/atb@latest`)

## Demo Steps

```bash
mkdir -p /tmp/atb-e2e-demo
cd /tmp/atb-e2e-demo

# 1) Initialize a new local bundle.
atb bundle new

# 2) Append workflow events.
atb append ai.request.received --data='{"request_id":"req-1001","channel":"support","actor":"agent-a"}'
atb append ai.model.invoked --data='{"provider":"openai","model":"gpt-5","purpose":"triage"}'
atb append ai.model.output --data='{"outcome":"allow","confidence":0.92}'
atb append ai.action.executed --data='{"action":"send_reply","ticket_id":"T-1042"}'

# 3) Add a named checkpoint.
atb snapshot e2e_demo_complete

# 4) Verify chain integrity.
atb verify

# 5) Produce human-readable trust output.
atb trust-report --format markdown
```

## Expected Outcome

- `run.atb/bundle.atb` exists and contains the appended records.
- `atb verify` reports a valid bundle hash chain.
- `atb trust-report` emits workflow evidence suitable for pilot demos and review.
