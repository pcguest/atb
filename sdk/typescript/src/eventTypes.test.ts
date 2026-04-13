import { describe, expect, it } from "vitest";
import {
  AI_ACTION_COMMITTED_EVENT_TYPE,
  AI_ACTION_EXECUTED_EVENT_TYPE,
  AI_ACTION_PRECOMMIT_EVENT_TYPE,
  AI_CHAIN_RUN_EVENT_TYPE,
  AI_HUMAN_APPROVAL_EVENT_TYPE,
  AI_JOB_COMPLETED_EVENT_TYPE,
  AI_JOB_SCHEDULED_EVENT_TYPE,
  AI_JOB_STARTED_EVENT_TYPE,
  AI_JOB_STEP_EVENT_TYPE,
  AI_LLM_CALL_EVENT_TYPE,
  AI_MODEL_INVOKED_EVENT_TYPE,
  AI_MODEL_OUTPUT_EVENT_TYPE,
  AI_OVERRIDE_REQUESTED_EVENT_TYPE,
  AI_POLICY_DECISION_EVENT_TYPE,
  AI_REQUEST_RECEIVED_EVENT_TYPE,
  AI_RESPONSE_SENT_EVENT_TYPE,
  AI_RETRIEVAL_EXECUTED_EVENT_TYPE,
  AI_TOOL_EXEC_EVENT_TYPE,
  BUNDLE_ANCHOR_EVENT_TYPE,
  BUNDLE_MANIFEST_EVENT_TYPE,
  BUNDLE_SIGNATURE_EVENT_TYPE,
  DATA_EXPORT_EXECUTED_EVENT_TYPE,
  DATA_EXPORT_PRECOMMIT_EVENT_TYPE,
  RAG_INDEX_EVENT_TYPE,
  RAG_RETRIEVAL_EVENT_TYPE,
  SNAPSHOT_EVENT_TYPE,
} from "./eventTypes.js";

describe("event type constants", () => {
  it("match the Go registry values", () => {
    expect(BUNDLE_MANIFEST_EVENT_TYPE).toBe("atb.bundle.manifest");
    expect(BUNDLE_ANCHOR_EVENT_TYPE).toBe("atb.bundle.anchor");
    expect(BUNDLE_SIGNATURE_EVENT_TYPE).toBe("atb.bundle.signature");
    expect(SNAPSHOT_EVENT_TYPE).toBe("atb.snapshot");
    expect(AI_REQUEST_RECEIVED_EVENT_TYPE).toBe("ai.request.received");
    expect(AI_RESPONSE_SENT_EVENT_TYPE).toBe("ai.response.sent");
    expect(AI_LLM_CALL_EVENT_TYPE).toBe("ai.llm.call");
    expect(AI_TOOL_EXEC_EVENT_TYPE).toBe("ai.tool.exec");
    expect(AI_CHAIN_RUN_EVENT_TYPE).toBe("ai.chain.run");
    expect(AI_POLICY_DECISION_EVENT_TYPE).toBe("ai.policy.decision");
    expect(AI_RETRIEVAL_EXECUTED_EVENT_TYPE).toBe("ai.retrieval.executed");
    expect(AI_MODEL_INVOKED_EVENT_TYPE).toBe("ai.model.invoked");
    expect(AI_MODEL_OUTPUT_EVENT_TYPE).toBe("ai.model.output");
    expect(RAG_INDEX_EVENT_TYPE).toBe("atb.event.rag_index");
    expect(RAG_RETRIEVAL_EVENT_TYPE).toBe("atb.event.rag_retrieval");
    expect(AI_ACTION_PRECOMMIT_EVENT_TYPE).toBe("ai.action.precommit");
    expect(AI_ACTION_EXECUTED_EVENT_TYPE).toBe("ai.action.executed");
    expect(AI_ACTION_COMMITTED_EVENT_TYPE).toBe("ai.action.committed");
    expect(AI_HUMAN_APPROVAL_EVENT_TYPE).toBe("ai.human.approval");
    expect(AI_OVERRIDE_REQUESTED_EVENT_TYPE).toBe("ai.override.requested");
    expect(AI_JOB_SCHEDULED_EVENT_TYPE).toBe("ai.job.scheduled");
    expect(AI_JOB_STARTED_EVENT_TYPE).toBe("ai.job.started");
    expect(AI_JOB_STEP_EVENT_TYPE).toBe("ai.job.step");
    expect(AI_JOB_COMPLETED_EVENT_TYPE).toBe("ai.job.completed");
    expect(DATA_EXPORT_PRECOMMIT_EVENT_TYPE).toBe("data.export.precommit");
    expect(DATA_EXPORT_EXECUTED_EVENT_TYPE).toBe("data.export.executed");
  });
});
