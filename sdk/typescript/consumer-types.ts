import {
  AI_CONTEXT_OPERATION,
  AI_CONTEXT_UNIT,
  AI_REQUEST_RECEIVED_EVENT_TYPE,
  ATB_MCP_OPERATION,
  Bundle,
  SDK_VERSION,
} from "./dist/index.js";

const version: string = SDK_VERSION;
const bundle = new Bundle();
bundle.append(AI_REQUEST_RECEIVED_EVENT_TYPE, {
  request_id: "req-package-types",
  actor_id_hash: "sha256:package-types",
  purpose_tag: "release_acceptance",
});

void version;
void bundle;
// These schema-generated v1.16 constants must remain exported by the packed
// package entry point, not only by its internal generated module.
void AI_CONTEXT_UNIT;
void AI_CONTEXT_OPERATION;
void ATB_MCP_OPERATION;
