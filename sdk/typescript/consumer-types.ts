import {
  AI_REQUEST_RECEIVED_EVENT_TYPE,
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
