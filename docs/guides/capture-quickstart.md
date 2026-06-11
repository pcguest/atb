# Capture quickstart

This guide shows a narrow Capture v1 path from a saved chatlog to a verified local bundle.

It keeps the existing ATB model intact:

- one local bundle on disk
- one local viewer for that bundle
- the same `atb verify` profiles and CAS output
- no hosted control plane and no claim of complete capture

## From saved chatlog to verified bundle in 5 minutes

```bash
go install -tags noembed github.com/pcguest/atb/cmd/atb@latest
atb import chatlog --from generic-jsonl --input testdata/chatlog.jsonl --snapshot imported_chatlog
atb verify --bundle run.atb/bundle.atb --profile atb.profile.rag_answer --format json
atb view --bundle run.atb/bundle.atb
```

What this does:

1. imports a saved local chatlog file into canonical ATB events
2. writes those events into `run.atb/bundle.atb`
3. labels the end state with `imported_chatlog`
4. verifies the resulting bundle against the RAG profile
5. opens the same single-bundle local review UI used elsewhere in ATB

## The example file

The repository ships a small example chatlog at [`testdata/chatlog.jsonl`](../../testdata/chatlog.jsonl).

See the full generic JSONL schema in [Chatlog import](../integrations/chatlog-import.md).

## Capture wrapper

`atb capture run` is the other Capture v1 entry point.

It does not proxy provider traffic or auto-attach to arbitrary runtimes. It prepares a bundle path, sets local environment variables for the child process, and then runs the child command.

Example:

```bash
atb capture run --env-prefix MYAPP -- ./agent-runner --config ./agent.yaml
```

By default the child sees:

- `ATB_BUNDLE_PATH`
- `ATB_CAPTURE_RUN_ID`
- `ATB_CAPTURE_MODE=run`

When `--env-prefix MYAPP` is supplied, the child also sees:

- `MYAPP_BUNDLE_PATH`
- `MYAPP_CAPTURE_RUN_ID`
- `MYAPP_CAPTURE_MODE=run`

If the child can already write ATB events, point it at `ATB_BUNDLE_PATH` or the prefixed equivalent. If not, a practical Capture v1 pattern is to have the child write a JSONL trace that you later import with `atb import chatlog`.

Optional verification after a successful child run:

```bash
atb capture run --profile atb.profile.rag_answer -- ./agent-runner
```

When `--profile` is supplied, `capture run` verifies the resulting bundle after the child exits successfully. A non-zero child exit still wins: the wrapper returns the child exit code unless the capture layer itself hits a fatal error.

## Boundaries to keep in mind

- Capture v1 reduces manual event entry. It does not guarantee that every relevant event was captured.
- CAS still describes recorded evidence completeness within the selected profile boundary.
- ATB still does not provide certification, legal advice, or a hosted workspace story.
