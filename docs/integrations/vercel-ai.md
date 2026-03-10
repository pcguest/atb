# Vercel AI SDK Integration

ATB includes a middleware helper for emitting `ai.llm.call`, `ai.tool.exec`, and `ai.chain.run` events from Vercel AI SDK style callbacks.

## Install

```bash
npm install @pcguest/atb-sdk ai
```

This uses the same TypeScript package listed in the README installation options.

## Quick Start

```ts
import { Bundle, atbMiddleware } from "@pcguest/atb-sdk";
import { streamText } from "ai";
import { openai } from "@ai-sdk/openai";

const bundle = new Bundle();

// privacyMode: "off" | "hash" | "redact"
const trace = atbMiddleware({
  bundle,
  privacyMode: "redact",
  autoSave: true,
  savePath: "run.atb/bundle.atb",
});

trace.onChainStart({ runId: "chain-1", chainName: "chat", inputs: { prompt: "hello" } });
trace.onLLMStart({ runId: "llm-1", parentRunId: "chain-1", provider: "openai", model: "gpt-4o-mini", prompt: "hello" });

const result = streamText({
  model: openai("gpt-4o-mini"),
  prompt: "hello",
  onToken(token) {
    trace.onTokenGenerated(token, "llm-1");
  },
  onStepFinish(step) {
    trace.onStepFinish({
      runId: "llm-1",
      text: step.text,
      usage: {
        promptTokens: step.usage?.promptTokens,
        completionTokens: step.usage?.completionTokens,
        totalTokens: step.usage?.totalTokens,
      },
      finishReason: step.finishReason,
    });
    trace.onChainEnd({ runId: "chain-1", outputs: { text: step.text } });
  },
});

await result.text;
```

## Disable Tracing

```ts
const trace = atbMiddleware({ enabled: false });
```

The middleware remains callable and will no-op safely.
