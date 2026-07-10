#!/usr/bin/env bash
set -euo pipefail

required_files=(
  "docs/integrations/reference-agent-integrations.md"
  "examples/python/agent_incident_demo.py"
  "examples/python/langgraph_demo.py"
  "examples/python/langchain_bot.py"
  "examples/typescript/vercel-chat-bot.ts"
  "sdk/python/tests/test_sdk_capture.py"
  "sdk/typescript/src/sdk-capture.test.ts"
  "docs/integrations/README.md"
)

for path in "${required_files[@]}"; do
  test -f "$path" || {
    echo "missing reference integration file: $path" >&2
    exit 1
  }
done

doc="docs/integrations/reference-agent-integrations.md"
grep -Fq "No external credentials" "$doc"
grep -Fiq "fake-provider tests" "$doc"
grep -Fq "Capture completeness is bounded" "$doc"
grep -Fq "agent_incident_demo.py" "$doc"
grep -Fq "vercel-chat-bot.ts" "$doc"
grep -Fq "langchain_bot.py" "$doc"
grep -Fq "langgraph_demo.py" "$doc"
grep -Fq "reference-agent-integrations.md" docs/integrations/README.md

echo "ok: reference integration docs are present and linked"
