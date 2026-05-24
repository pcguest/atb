# Launch demo assets

Regenerate visual assets before a public launch. Paths referenced from
[README.md](../../README.md) and [docs/quickstart.md](../quickstart.md).

| Asset | Purpose |
|-------|---------|
| `atb-verify-demo.gif` | Terminal walkthrough: bundle init, append, verify pass, tamper, verify fail |
| `atb-verify-report.png` | Example `atb verify --format json` summary panel |

## Record the GIF

Follow [docs/guides/tamper-demo.md](../guides/tamper-demo.md), then record the
terminal session (for example with `asciinema` + `agg`, or screen capture).

## Capture the report PNG

```bash
go build -o ./atb ./cmd/atb
bash examples/quickstart/run.sh > /tmp/atb-verify.txt
./atb verify --profile atb.profile.privileged_tool_action --format json | tee /tmp/atb-verify.json
```

Screenshot the JSON output or viewer Profile CAS panel after `atb view`.

Place files in this directory and verify README links.
