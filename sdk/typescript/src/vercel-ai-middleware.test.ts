import { createHash } from "node:crypto";
import { describe, expect, it } from "vitest";
import { Bundle } from "./bundle.js";
import { atbMiddleware } from "./vercel-ai-middleware.js";

function sha256(value: string): string {
  return createHash("sha256").update(value, "utf8").digest("hex");
}

function userRecords(bundle: Bundle) {
  return bundle.records.filter(
    (record) => record.event.type !== "atb.bundle.manifest"
  );
}

describe("atbMiddleware", () => {
  it("maps chain, llm streaming, and tool events", () => {
    const bundle = new Bundle();
    const middleware = atbMiddleware({ bundle, privacyMode: "off" });

    middleware.onChainStart({ runId: "chain-1", chainName: "qa_chain", inputs: { query: "what is atb" } });
    middleware.onLLMStart({
      runId: "llm-1",
      parentRunId: "chain-1",
      provider: "openai",
      model: "gpt-4o-mini",
      prompt: "hello",
    });
    middleware.onTokenGenerated("A", "llm-1");
    middleware.onTokenGenerated("B", "llm-1");
    middleware.onStepFinish({
      runId: "llm-1",
      usage: { promptTokens: 3, completionTokens: 2, totalTokens: 5 },
      finishReason: "stop",
      toolCalls: [{ toolName: "weather.lookup", input: { city: "Melbourne" }, output: { temp_c: 24 } }],
    });
    middleware.onChainEnd({ runId: "chain-1", outputs: { answer: "AB" } });

    const records = userRecords(bundle);
    const types = records.map((record) => record.event.type);
    expect(types).toEqual([
      "ai.chain.run",
      "ai.llm.call",
      "ai.request.received",
      "ai.model.invoked",
      "ai.llm.call",
      "ai.llm.call",
      "ai.tool.exec",
      "ai.tool.exec",
      "ai.llm.call",
      "ai.model.output",
      "ai.chain.run",
    ]);

    const chainStart = records[0].event;
    const llmStart = records[1].event;
    const request = records[2].event.data as Record<string, unknown>;
    const invoked = records[3].event.data as Record<string, unknown>;
    const llmEnd = records[8].event.data as Record<string, unknown>;
    const output = records[9].event.data as Record<string, unknown>;

    expect(llmStart.trace_id).toBe(chainStart.trace_id);
    expect(llmStart.parent_span_id).toBe(chainStart.span_id);
    expect(request.request_id).toBe("llm-1");
    expect(String(request.input_digest).startsWith("sha256:")).toBe(true);
    expect(invoked.request_id).toBe("llm-1");
    expect(invoked.model_provider).toBe("openai");
    expect(invoked.model_id).toBe("gpt-4o-mini");
    expect(String(invoked.model_parameters_digest).startsWith("sha256:")).toBe(true);

    const endContext = llmEnd.context as Record<string, unknown>;
    const completion = endContext.completion as Record<string, unknown>;
    expect(completion.text).toBe("AB");

    const usage = endContext.token_usage as Record<string, unknown>;
    expect(usage.total_tokens).toBe(5);
    expect(output.request_id).toBe("llm-1");
    expect(String(output.output_digest).startsWith("sha256:")).toBe(true);
    expect(output.output_format).toBe("text");
  });

  it("applies hash privacy mode and hashes emitted text", () => {
    const bundle = new Bundle();
    const middleware = atbMiddleware({ bundle, privacyMode: "hash" });

    middleware.onLLMStart({ runId: "llm-hash", prompt: "secret prompt" });

    const payload = userRecords(bundle)[0].event.data as Record<string, unknown>;
    const context = payload.context as Record<string, unknown>;
    const prompt = context.prompt as Record<string, string>;

    expect(prompt.text.startsWith("sha256:")).toBe(true);
    expect(prompt.sha256).toBe(`sha256:${sha256(prompt.text)}`);
  });

  it("applies redact privacy mode and hashes redacted text", () => {
    const bundle = new Bundle();
    const middleware = atbMiddleware({ bundle, privacyMode: "redact" });

    middleware.onLLMStart({ runId: "llm-redact", prompt: "secret prompt" });

    const payload = userRecords(bundle)[0].event.data as Record<string, unknown>;
    const context = payload.context as Record<string, unknown>;
    const prompt = context.prompt as Record<string, string>;

    expect(prompt.text).toBe("[REDACTED]");
    expect(prompt.sha256).toBe(`sha256:${sha256("[REDACTED]")}`);
  });

  it("no-ops when disabled", () => {
    const bundle = new Bundle();
    const middleware = atbMiddleware({ bundle, enabled: false });

    middleware.onChainStart({ runId: "chain-1", chainName: "qa_chain", inputs: { query: "x" } });
    middleware.onLLMStart({ runId: "llm-1", prompt: "hello" });

    expect(bundle.records).toHaveLength(1);
  });
});
