import type {
  APIError,
  BundleEventsResponse,
  BundleGraphResponse,
  BundleMetaResponse,
  PrivacyRevealRequest,
  PrivacyRevealResponse,
  VerificationResponse,
} from "@/lib/types";

async function requestJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {}),
    },
    cache: "no-store",
  });

  if (!response.ok) {
    let message = `Request failed (${response.status})`;
    try {
      const payload = (await response.json()) as APIError;
      if (payload?.error) {
        message = payload.error;
      }
    } catch {
      // keep fallback message
    }
    throw new Error(message);
  }

  return (await response.json()) as T;
}

export function getVerification(): Promise<VerificationResponse> {
  return requestJSON<VerificationResponse>("/api/v1/verification");
}

export function getBundleMeta(): Promise<BundleMetaResponse> {
  return requestJSON<BundleMetaResponse>("/api/v1/bundle/meta");
}

export function getBundleEvents(offset = 0, limit = 200): Promise<BundleEventsResponse> {
  const params = new URLSearchParams({
    offset: String(offset),
    limit: String(limit),
  });
  return requestJSON<BundleEventsResponse>(`/api/v1/bundle/events?${params.toString()}`);
}

export function getBundleGraph(): Promise<BundleGraphResponse> {
  return requestJSON<BundleGraphResponse>("/api/v1/bundle/graph");
}

export function revealField(payload: PrivacyRevealRequest): Promise<PrivacyRevealResponse> {
  return requestJSON<PrivacyRevealResponse>("/api/v1/privacy/reveal", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}
