import { describe, expect, it } from "vitest";
import { AgentClient, type AgentRequestFn } from "./agent-client.js";
import { Bundle } from "./bundle.js";
import { AutomationSession } from "./automation-session.js";
import { isCaptureEnvironment } from "./capture.js";

function userTypes(bundle: Bundle): string[] {
  return bundle.records
    .filter((record) => record.event.type !== "atb.bundle.manifest")
    .map((record) => record.event.type);
}

describe("AutomationSession", () => {
  it("records a chained RAG workflow on one request", () => {
    const session = new AutomationSession({
      bundle: new Bundle(),
      actorId: "actor-rag",
      purposeTag: "rag_answer",
      requestId: "req-chain-1",
    });

    session.logRetrieval({
      query: "what is ATB?",
      corpusId: "docs-v1",
      corpusVersion: "2026-05",
      topK: 3,
      resultSet: [{ id: "doc-1" }],
    });
    session.logModelInvocation({
      provider: "openai",
      model: "gpt-4o",
      prompt: "Answer using context",
      parameters: { temperature: 0 },
    });
    session.logModelOutput({ output: "ATB is an audit substrate." });
    session.logResponseSent({ output: "ATB is an audit substrate." });

    expect(userTypes(session.bundle)).toEqual([
      "ai.request.received",
      "ai.retrieval.executed",
      "ai.model.invoked",
      "ai.model.output",
      "ai.response.sent",
    ]);

    const request = session.bundle.records.find(
      (r) => r.event.type === "ai.request.received"
    )?.event.data as Record<string, unknown>;
    expect(request.request_id).toBe("req-chain-1");
    expect(request.purpose_tag).toBe("rag_answer");
  });

  it("records privileged tool action with commit", async () => {
    const session = new AutomationSession({ bundle: new Bundle(), actorId: "actor-tool" });

    await session.runToolAction(
      {
        actionType: "restart_service",
        targetResourceId: "svc-api",
        intendedEffect: "roll restart",
        actionParameters: { service: "api" },
        actionId: "act-tool-1",
      },
      () => ({ status: "restarted" })
    );

    expect(userTypes(session.bundle)).toEqual([
      "ai.request.received",
      "ai.action.precommit",
      "ai.policy.decision",
      "ai.action.executed",
      "ai.action.committed",
    ]);
  });

  it("opens from capture environment when bundle path is set", () => {
    const env = {
      ATB_BUNDLE_PATH: "/tmp/run.atb/bundle.atb",
      ATB_CAPTURE_RUN_ID: "cap-deadbeef",
      ATB_CAPTURE_MODE: "run",
    };
    expect(isCaptureEnvironment(env)).toBe(true);

    const session = AutomationSession.fromCaptureEnvironment(env);
    expect(session).not.toBeNull();
    expect(session!.captureRunId).toBe("cap-deadbeef");
    expect(session!.bundle).toBeDefined();
  });

  it("returns null outside capture environment", () => {
    expect(AutomationSession.fromCaptureEnvironment({})).toBeNull();
  });

  it("appends snapshot on close", () => {
    const session = new AutomationSession({ bundle: new Bundle() });
    session.beginRequest("policy_decision");
    session.logPolicyDecision(
      {
        actionType: "approve",
        actionParameters: { ticket: "INC-9" },
      },
      { decision: "allow", reasonCodes: ["ok"] }
    );
    session.snapshot("review_boundary");

    expect(userTypes(session.bundle)).toContain("atb.snapshot");
  });

  it("falls back to local bundle when agent is absent", () => {
    const session = new AutomationSession({
      bundle: new Bundle(),
      env: {},
      agentClient: null,
    });
    expect(session.isUsingAgent()).toBe(false);
    session.logModelInvocation({
      provider: "openai",
      model: "gpt-4o",
      prompt: "hi",
    });
    expect(userTypes(session.bundle)).toContain("ai.model.invoked");
  });

  it("routes workflow events through the agent when configured", () => {
    const events: Array<{ eventType: string; payload: Record<string, unknown> }> =
      [];
    const requestFn: AgentRequestFn = (url, init) => {
      if (url.endsWith("/v1/session/open")) {
        return {
          status: 201,
          body: JSON.stringify({
            session_id: "sess_agent",
            bundle_path: "/tmp/agent/bundle.atb",
          }),
        };
      }
      if (url.includes("/event")) {
        const parsed = JSON.parse(init.body ?? "{}") as {
          event_type: string;
          payload: Record<string, unknown>;
        };
        events.push({
          eventType: parsed.event_type,
          payload: parsed.payload,
        });
        return { status: 202, body: JSON.stringify({ status: "queued" }) };
      }
      if (url.includes("/close")) {
        return {
          status: 200,
          body: JSON.stringify({
            session_id: "sess_agent",
            bundle_path: "/tmp/agent/bundle.atb",
            head_hash: "deadbeef",
            event_count: events.length,
            opened_at: "2026-05-25T12:00:00Z",
            closed_at: "2026-05-25T12:01:00Z",
          }),
        };
      }
      return { status: 404, body: JSON.stringify({ error: "not found" }) };
    };

    const agentClient = new AgentClient({
      baseUrl: "http://127.0.0.1:6180",
      requestFn,
    });
    const session = new AutomationSession({
      agentClient,
      actorId: "actor-agent",
      purposeTag: "rag_answer",
      requestId: "req-agent-1",
    });

    expect(session.isUsingAgent()).toBe(true);
    session.logRetrieval({
      query: "what is ATB?",
      corpusId: "docs-v1",
      corpusVersion: "2026-05",
      topK: 3,
      resultSet: [{ id: "doc-1" }],
    });
    session.logModelInvocation({
      provider: "openai",
      model: "gpt-4o",
      prompt: "Answer using context",
    });
    session.close();

    expect(events.map((event) => event.eventType)).toEqual([
      "ai.request.received",
      "ai.retrieval.executed",
      "ai.model.invoked",
    ]);
    expect(events[0]?.payload.purpose_tag).toBe("rag_answer");
    expect(userTypes(session.bundle)).toEqual([]);
  });
});
