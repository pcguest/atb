import { Bundle, atbMiddleware } from "@pcguest/atb-sdk";
import { streamText } from "ai";
import { openai } from "@ai-sdk/openai";

async function main(): Promise<void> {
  const apiKey = process.env.OPENAI_API_KEY;
  if (!apiKey) {
    throw new Error("OPENAI_API_KEY is required");
  }

  const bundle = new Bundle();

  // Toggle privacyMode: "off" | "hash" | "redact"
  const trace = atbMiddleware({
    bundle,
    privacyMode: "redact",
    autoSave: true,
    savePath: "run.atb/bundle.atb",
    framework: "vercel-ai",
  });

  trace.onChainStart({
    runId: "chain-1",
    chainName: "chat-bot",
    inputs: { prompt: "Summarize why ATB is useful for agent audit trails." },
  });

  trace.onLLMStart({
    runId: "llm-1",
    parentRunId: "chain-1",
    provider: "openai",
    model: "gpt-4o-mini",
    prompt: "Summarize why ATB is useful for agent audit trails.",
  });

  const result = streamText({
    model: openai("gpt-4o-mini", { apiKey }),
    prompt: "Summarize why ATB is useful for agent audit trails.",
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
    },
  });

  const text = await result.text;

  trace.onChainEnd({ runId: "chain-1", outputs: { text } });

  const path = bundle.save("run.atb/bundle.atb");
  console.log(text);
  console.log(`ATB bundle written to: ${path}`);
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
