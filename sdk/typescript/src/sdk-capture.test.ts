import { describe, expect, it } from "vitest";
import { Bundle } from "./bundle.js";
import { wrapAnthropic, wrapOpenAI } from "./sdk-capture.js";

function userRecords(bundle: Bundle) {
  return bundle.records.filter((record) => record.event.type !== "atb.bundle.manifest");
}

describe("wrapOpenAI", () => {
  it("records request, model, and output for a non-streaming chat call", async () => {
    const bundle = new Bundle();
    const fakeCreate = async () => ({
      model: "gpt-4o",
      choices: [
        {
          message: { content: "hello back" },
          finish_reason: "stop",
        },
      ],
      usage: { prompt_tokens: 5, completion_tokens: 2, total_tokens: 7 },
    });
    const create = wrapOpenAI(fakeCreate, { bundle, privacyMode: "off" });

    const res = await create({ model: "gpt-4o", messages: [{ role: "user", content: "hi" }] });
    expect(res.choices?.[0]?.message?.content).toBe("hello back");

    const types = userRecords(bundle).map((r) => r.event.type);
    expect(types).toEqual([
      "atb.capture.scope",
      "ai.llm.call",
      "ai.request.received",
      "ai.model.invoked",
      "ai.llm.call",
      "ai.model.output",
    ]);

    const scope = userRecords(bundle)[0].event.data as Record<string, unknown>;
    expect(scope.targets).toEqual(["openai"]);
    expect(scope.capture_mode).toBe("raw");

    const invoked = userRecords(bundle)[3].event.data as Record<string, unknown>;
    expect(invoked.model_provider).toBe("openai");
    expect(invoked.model_id).toBe("gpt-4o");

    const end = userRecords(bundle)[4].event.data as Record<string, unknown>;
    const context = end.context as Record<string, unknown>;
    const completion = context.completion as Record<string, string>;
    expect(completion.text).toBe("hello back");
    const usage = context.token_usage as Record<string, number>;
    expect(usage.total_tokens).toBe(7);
  });

  it("records tool calls from the response", async () => {
    const bundle = new Bundle();
    const fakeCreate = async () => ({
      choices: [
        {
          message: {
            content: "",
            tool_calls: [{ function: { name: "weather.lookup", arguments: '{"city":"Melbourne"}' } }],
          },
          finish_reason: "tool_calls",
        },
      ],
    });
    const create = wrapOpenAI(fakeCreate, { bundle });
    await create({ model: "gpt-4o", messages: [{ role: "user", content: "weather?" }] });

    const types = userRecords(bundle).map((r) => r.event.type);
    expect(types).toContain("ai.tool.exec");
    const toolExec = userRecords(bundle).find((r) => r.event.type === "ai.tool.exec")!
      .event.data as Record<string, unknown>;
    const toolContext = toolExec.context as Record<string, unknown>;
    expect(toolContext.tool_name).toBe("weather.lookup");
  });

  it("records an error event and rethrows", async () => {
    const bundle = new Bundle();
    const boom = new Error("rate limited");
    const create = wrapOpenAI(
      async () => {
        throw boom;
      },
      { bundle },
    );
    await expect(create({ model: "gpt-4o", messages: [] })).rejects.toThrow("rate limited");

    const errorRecord = userRecords(bundle).find((r) => {
      const data = r.event.data as Record<string, unknown>;
      const status = data.status as Record<string, unknown> | undefined;
      return status && status.ok === false;
    });
    expect(errorRecord).toBeDefined();
  });

  it("hashes the prompt under hash privacy mode", async () => {
    const bundle = new Bundle();
    const create = wrapOpenAI(async () => ({ choices: [{ message: { content: "x" } }] }), {
      bundle,
      privacyMode: "hash",
    });
    await create({ model: "gpt-4o", messages: [{ role: "user", content: "secret" }] });

    const scope = userRecords(bundle)[0].event.data as Record<string, unknown>;
    expect(scope.capture_mode).toBe("digest");

    const start = userRecords(bundle)[1].event.data as Record<string, unknown>;
    const context = start.context as Record<string, unknown>;
    const prompt = context.prompt as Record<string, string>;
    expect(prompt.text.startsWith("sha256:")).toBe(true);
    expect(prompt.text).not.toContain("secret");
  });

  it("is a no-op recorder but still calls through when disabled", async () => {
    const bundle = new Bundle();
    let called = false;
    const create = wrapOpenAI(
      async () => {
        called = true;
        return { choices: [{ message: { content: "y" } }] };
      },
      { bundle, enabled: false },
    );
    const res = await create({ model: "gpt-4o", messages: [] });
    expect(called).toBe(true);
    expect(res.choices?.[0]?.message?.content).toBe("y");
    expect(userRecords(bundle)).toHaveLength(0);
  });

  it("throws on streaming requests", async () => {
    const create = wrapOpenAI(async () => ({}), {});
    await expect(create({ model: "gpt-4o", messages: [], stream: true })).rejects.toThrow(
      /streaming/,
    );
  });
});

describe("wrapAnthropic", () => {
  it("records text and tool_use blocks from a messages response", async () => {
    const bundle = new Bundle();
    const fakeCreate = async () => ({
      model: "claude-sonnet-4-6",
      content: [
        { type: "text", text: "the answer is " },
        { type: "text", text: "42" },
        { type: "tool_use", name: "calc", input: { expr: "6*7" } },
      ],
      usage: { input_tokens: 10, output_tokens: 4 },
      stop_reason: "end_turn",
    });
    const create = wrapAnthropic(fakeCreate, { bundle });

    await create({
      model: "claude-sonnet-4-6",
      max_tokens: 256,
      system: "be terse",
      messages: [{ role: "user", content: "what is 6*7" }],
    });

    const records = userRecords(bundle);
    const invoked = records.find((r) => r.event.type === "ai.model.invoked")!.event.data as Record<
      string,
      unknown
    >;
    expect(invoked.model_provider).toBe("anthropic");
    expect(invoked.model_id).toBe("claude-sonnet-4-6");

    const end = records.filter((r) => r.event.type === "ai.llm.call").at(-1)!.event.data as Record<
      string,
      unknown
    >;
    const context = end.context as Record<string, unknown>;
    const completion = context.completion as Record<string, string>;
    expect(completion.text).toBe("the answer is 42");
    const usage = context.token_usage as Record<string, number>;
    expect(usage.prompt_tokens).toBe(10);
    expect(usage.completion_tokens).toBe(4);
    expect(usage.total_tokens).toBe(14);

    const tool = records.find((r) => r.event.type === "ai.tool.exec")!.event.data as Record<
      string,
      unknown
    >;
    const toolContext = tool.context as Record<string, unknown>;
    expect(toolContext.tool_name).toBe("calc");
  });

  it("records an error and rethrows", async () => {
    const bundle = new Bundle();
    const create = wrapAnthropic(
      async () => {
        throw new Error("overloaded");
      },
      { bundle },
    );
    await expect(create({ model: "claude-sonnet-4-6", messages: [] })).rejects.toThrow("overloaded");
    const errorRecord = userRecords(bundle).find((r) => {
      const status = (r.event.data as Record<string, unknown>).status as
        | Record<string, unknown>
        | undefined;
      return status && status.ok === false;
    });
    expect(errorRecord).toBeDefined();
  });

  it("throws on streaming requests", async () => {
    const create = wrapAnthropic(async () => ({}), {});
    await expect(create({ model: "claude-sonnet-4-6", messages: [], stream: true })).rejects.toThrow(
      /streaming/,
    );
  });
});
