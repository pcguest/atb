# Mortise custody integration

Mortise is the optional commercial custody and organisational layer for ATB
evidence. Production storage, API, identity, retention, and operator controls
are versioned with Mortise and are not defined by this repository.

ATB does not implement WORM storage or a custody daemon.

## Authentication

Keep the Bearer token in `ATB_MORTISE_TOKEN` or a local secret manager. Do not
pass credentials on the command line or commit them to scripts. The legacy
`ATB_CUSTOS_TOKEN` name remains a deprecated fallback.

```sh
export ATB_MORTISE_TOKEN="<mortise-api-key>"
export MORTISE_URL="https://mortise.example"
```

## Capture auto-push

`atb intercept` lodges the closed bundle after each session:

```sh
atb intercept \
  --port 8080 \
  --bundle /tmp/agent.atb \
  --target openai,anthropic \
  --mortise "$MORTISE_URL"
```

Capture remains fully local when `--mortise` is absent.

## Incident custody

Lodge the authoritative bundle and print its signed receipt:

```sh
atb incident export \
  --bundle /tmp/agent.atb \
  --session sess_abc123 \
  --mortise-endpoint "$MORTISE_URL"
```

Use `--out incident.zip` instead for a local-only incident package. These
destinations are mutually exclusive.

## Compliance pack with custody proof

Create an offline compliance pack, lodge its authoritative bundle, and include
the returned receipt under manifest/checksum coverage:

```sh
atb compliance pack \
  --bundle /tmp/agent.atb \
  --profile atb.profile.privileged_tool_action \
  --regime eu-ai-act \
  --out compliance.zip \
  --mortise-endpoint "$MORTISE_URL"
```

The resulting pack contains `mortise/receipt.json`. Mortise custodies the
bundle, not the derived ZIP.

## Compatibility aliases

`--custos`, `--custos-endpoint`, and `ATB_CUSTOS_TOKEN` remain accepted as
deprecated aliases. New documentation and automation must use Mortise names.
