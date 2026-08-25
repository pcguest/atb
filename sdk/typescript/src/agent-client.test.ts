import { spawn } from "node:child_process";
import { once } from "node:events";
import { describe, expect, it } from "vitest";
import {
  AgentClient,
  AgentClientError,
  DEFAULT_AGENT_URL,
  MAX_AGENT_RESPONSE_BYTES,
  isExplicitAgentUrl,
  probeAgentHealth,
  resolveAgentBaseUrl,
  syncAgentRequest,
  tryCreateAgentClient,
  type AgentRequestFn,
} from "./agent-client.js";

function mockRequest(
  handler: (url: string, init: { method: string; body?: string }) => AgentRequestFn extends (
    ...args: infer A
  ) => infer R
    ? R
    : never
): AgentRequestFn {
  return (url, init) => handler(url, init);
}

describe("resolveAgentBaseUrl", () => {
  it("returns null when agent is disabled", () => {
    expect(resolveAgentBaseUrl({ ATB_AGENT_DISABLE: "1" })).toBeNull();
  });

  it("returns explicit ATB_AGENT_URL", () => {
    expect(resolveAgentBaseUrl({ ATB_AGENT_URL: "http://127.0.0.1:9999/" })).toBe(
      "http://127.0.0.1:9999"
    );
  });

  it("returns default URL when ATB_AGENT_AUTO is set", () => {
    expect(resolveAgentBaseUrl({ ATB_AGENT_AUTO: "1" })).toBe(DEFAULT_AGENT_URL);
  });
});

describe("AgentClient", () => {
  it("rejects non-loopback agent URLs", () => {
    expect(() => new AgentClient({ baseUrl: "https://agent.example.test" })).toThrow(
      /loopback/
    );
  });

  it("opens, appends, and closes a session", () => {
    const calls: Array<{ url: string; method: string; body?: string }> = [];
    const requestFn = mockRequest((url, init) => {
      calls.push({ url, method: init.method, body: init.body });
      if (url.endsWith("/healthz")) {
        return { status: 200, body: JSON.stringify({ status: "ok" }) };
      }
      if (url.endsWith("/v1/session/open")) {
        return {
          status: 201,
          body: JSON.stringify({
            session_id: "sess_test",
            bundle_path: "/tmp/agent/bundle.atb",
            actor_id: "actor-1",
          }),
        };
      }
      if (url.includes("/event")) {
        return { status: 202, body: JSON.stringify({ status: "queued" }) };
      }
      if (url.includes("/close")) {
        return {
          status: 200,
          body: JSON.stringify({
            session_id: "sess_test",
            bundle_path: "/tmp/agent/bundle.atb",
            head_hash: "abc123",
            event_count: 2,
            opened_at: "2026-05-25T12:00:00Z",
            closed_at: "2026-05-25T12:01:00Z",
          }),
        };
      }
      return { status: 404, body: JSON.stringify({ error: "not found" }) };
    });

    const client = new AgentClient({
      baseUrl: "http://127.0.0.1:6180",
      requestFn,
    });
    const opened = client.openSession({
      actorId: "actor-1",
      purposeTag: "rag_answer",
      bundlePath: "/tmp/agent/bundle.atb",
    });
    expect(opened.sessionId).toBe("sess_test");
    expect(opened.bundlePath).toBe("/tmp/agent/bundle.atb");

    client.appendEvent({
      eventType: "ai.model.invoked",
      payload: { request_id: "req-1" },
    });
    const closed = client.closeSession({ snapshotName: "review_boundary" });
    expect(closed.eventCount).toBe(2);

    expect(calls[0]?.url).toBe("http://127.0.0.1:6180/v1/session/open");
    expect(JSON.parse(calls[0]!.body!)).toEqual({
      actor_id: "actor-1",
      purpose_tag: "rag_answer",
      bundle_path: "/tmp/agent/bundle.atb",
    });
    expect(calls[1]?.url).toBe(
      "http://127.0.0.1:6180/v1/session/sess_test/event"
    );
    expect(JSON.parse(calls[1]!.body!)).toEqual({
      event_type: "ai.model.invoked",
      payload: { request_id: "req-1" },
    });
    expect(calls[2]?.url).toBe(
      "http://127.0.0.1:6180/v1/session/sess_test/close"
    );
    expect(JSON.parse(calls[2]!.body!)).toEqual({
      snapshot_name: "review_boundary",
    });
  });

  it("maps HTTP errors to AgentClientError", () => {
    const requestFn = mockRequest((url) => {
      if (url.endsWith("/v1/session/open")) {
        return {
          status: 201,
          body: JSON.stringify({
            session_id: "sess_test",
            bundle_path: "/tmp/agent/bundle.atb",
          }),
        };
      }
      return {
        status: 404,
        body: JSON.stringify({ error: "session not found" }),
      };
    });
    const client = new AgentClient({
      baseUrl: "http://127.0.0.1:6180",
      requestFn,
    });
    client.openSession();
    expect(() =>
      client.appendEvent({ eventType: "ai.model.invoked", payload: {} })
    ).toThrow(AgentClientError);
  });
});

describe("syncAgentRequest", () => {
  it("completes real loopback I/O and caps the response", async () => {
    const server = spawn(
      process.execPath,
      [
        "-e",
        `const http = require("node:http");
         const server = http.createServer((req, res) => {
           if (req.url === "/large") return res.end(Buffer.alloc(${MAX_AGENT_RESPONSE_BYTES + 1}));
           res.setHeader("content-type", "application/json");
           res.end(JSON.stringify({status: "ok"}));
         });
         server.listen(0, "127.0.0.1", () => console.log(server.address().port));`,
      ],
      { stdio: ["ignore", "pipe", "pipe"] }
    );
    try {
      let stderr = "";
      server.stderr.on("data", (chunk) => {
        stderr += String(chunk);
      });
      let chunk: unknown;
      try {
        [chunk] = await Promise.race([
          once(server.stdout, "data"),
          once(server, "exit").then(() => {
            throw new Error(stderr || "loopback test server exited before ready");
          }),
        ]);
      } catch (error) {
        if (error instanceof Error && error.message.includes("listen EPERM")) {
          return;
        }
        throw error;
      }
      const port = Number(String(chunk).trim());
      expect(Number.isInteger(port)).toBe(true);
      expect(
        syncAgentRequest(`http://127.0.0.1:${port}/healthz`, {
          method: "GET",
          timeoutMs: 1_000,
        })
      ).toEqual({ status: 200, body: JSON.stringify({ status: "ok" }) });
      expect(() =>
        syncAgentRequest(`http://127.0.0.1:${port}/large`, {
          method: "GET",
          timeoutMs: 1_000,
        })
      ).toThrow(/response exceeds/);
    } finally {
      server.kill();
    }
  }, 10_000);
});

describe("tryCreateAgentClient", () => {
  it("returns null when auto mode and health check fails", () => {
    const requestFn = mockRequest(() => ({ status: 503, body: "" }));
    expect(
      tryCreateAgentClient({ ATB_AGENT_AUTO: "1" }, requestFn)
    ).toBeNull();
  });

  it("returns client when explicit URL is set without health check", () => {
    const requestFn = mockRequest(() => ({ status: 503, body: "" }));
    const client = tryCreateAgentClient(
      { ATB_AGENT_URL: "http://127.0.0.1:6180" },
      requestFn
    );
    expect(client).not.toBeNull();
    expect(isExplicitAgentUrl({ ATB_AGENT_URL: "http://127.0.0.1:6180" })).toBe(
      true
    );
  });

  it("probes health for auto mode", () => {
    const requestFn = mockRequest((url) => {
      if (url.endsWith("/healthz")) {
        return { status: 200, body: JSON.stringify({ status: "ok" }) };
      }
      return { status: 404, body: "" };
    });
    expect(probeAgentHealth("http://127.0.0.1:6180", requestFn)).toBe(true);
    expect(tryCreateAgentClient({ ATB_AGENT_AUTO: "1" }, requestFn)).not.toBeNull();
  });
});
