import type { WorkspaceBundleSummary } from "@/lib/schemas/workspace";

/** Build a viewer URL for a workspace bundle (static-export safe query-param routing). */
export function bundleViewHref(bundle: WorkspaceBundleSummary, agentMode: boolean): string {
  const params = new URLSearchParams();
  if (agentMode) {
    params.set("session_id", bundle.session_id);
  }
  if (bundle.bundle_path) {
    params.set("bundle_path", bundle.bundle_path);
  }
  const query = params.toString();
  return query ? `/view/?${query}` : "/view/";
}

/** Short display label for a bundle row (session id or truncated head hash). */
export function bundleDisplayId(bundle: WorkspaceBundleSummary): string {
  if (bundle.session_id) {
    return bundle.session_id;
  }
  if (bundle.head_hash && bundle.head_hash.length >= 12) {
    return `${bundle.head_hash.slice(0, 12)}…`;
  }
  return bundle.id;
}

export function formatWorkspaceTimestamp(value: string): string {
  const parsed = Date.parse(value);
  if (Number.isNaN(parsed)) {
    return value;
  }
  return new Date(parsed).toLocaleString();
}
