const MAX_RESPONSE_BYTES = 1 << 20;
const RECEIPT_VERSION = "custos.receipt.v1";

export type MortiseJSON = Record<string, unknown>;

export interface MortiseReceipt extends MortiseJSON {
  receipt_version: string;
  receipt_id: string;
  bundle_hash: string;
}

export interface MortiseClientOptions {
  token?: string;
  timeoutMs?: number;
  fetch?: typeof globalThis.fetch;
}

export class MortiseError extends Error {
  readonly status?: number;

  constructor(message: string, status?: number) {
    super(message);
    this.name = "MortiseError";
    this.status = status;
  }
}

export class MortiseClient {
  private readonly endpoint: string;
  private readonly token: string;
  private readonly timeoutMs: number;
  private readonly fetchImpl: typeof globalThis.fetch;

  constructor(endpoint: string, options: MortiseClientOptions = {}) {
    let parsed: URL;
    try {
      parsed = new URL(endpoint.trim());
    } catch {
      throw new TypeError("invalid Mortise endpoint");
    }
    if (
      (parsed.protocol !== "http:" && parsed.protocol !== "https:") ||
      !parsed.host ||
      parsed.username !== "" ||
      parsed.password !== "" ||
      parsed.search !== "" ||
      parsed.hash !== ""
    ) {
      throw new TypeError("invalid Mortise endpoint");
    }
    const timeoutMs = options.timeoutMs ?? 30_000;
    if (!Number.isFinite(timeoutMs) || timeoutMs <= 0) {
      throw new TypeError("timeoutMs must be positive");
    }
    this.endpoint = parsed.toString().replace(/\/$/, "");
    this.token = options.token?.trim() ?? "";
    this.timeoutMs = timeoutMs;
    this.fetchImpl = options.fetch ?? globalThis.fetch;
    if (typeof this.fetchImpl !== "function") {
      throw new TypeError("fetch implementation is required");
    }
  }

  async ingestBundle(bundle: Uint8Array): Promise<MortiseReceipt> {
    if (bundle.byteLength === 0) {
      throw new TypeError("bundle must not be empty");
    }
    const receipt = await this.request("POST", "/ingest", bundle, "application/octet-stream");
    if (
      receipt.receipt_version !== RECEIPT_VERSION ||
      typeof receipt.receipt_id !== "string" ||
      receipt.receipt_id === "" ||
      typeof receipt.bundle_hash !== "string" ||
      receipt.bundle_hash === ""
    ) {
      throw new MortiseError("Mortise returned an invalid custody receipt");
    }
    return receipt as MortiseReceipt;
  }

  async verifyBundle(bundle: Uint8Array): Promise<MortiseJSON> {
    if (bundle.byteLength === 0) {
      throw new TypeError("bundle must not be empty");
    }
    return this.request("POST", "/verify/bundle", bundle, "application/octet-stream");
  }

  async verifyReceipt(receipt: MortiseJSON): Promise<MortiseJSON> {
    return this.request(
      "POST",
      "/verify/receipt",
      JSON.stringify(receipt),
      "application/json",
    );
  }

  async receiptsByHash(bundleHash: string): Promise<MortiseJSON> {
    const value = bundleHash.trim();
    if (value === "") {
      throw new TypeError("bundleHash must not be empty");
    }
    return this.request(
      "GET",
      `/receipts/by-hash?bundle_hash=${encodeURIComponent(value)}`,
    );
  }

  private async request(
    method: string,
    path: string,
    body?: Uint8Array | string,
    contentType?: string,
  ): Promise<MortiseJSON> {
    const headers = new Headers({ Accept: "application/json" });
    if (contentType) {
      headers.set("Content-Type", contentType);
    }
    if (this.token) {
      headers.set("Authorization", `Bearer ${this.token}`);
    }
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), this.timeoutMs);
    let response: Response;
    try {
      response = await this.fetchImpl(this.endpoint + path, {
        method,
        headers,
        body,
        redirect: "error",
        signal: controller.signal,
      });
    } catch (error) {
      throw new MortiseError(
        `Mortise request failed: ${error instanceof Error ? error.message : String(error)}`,
      );
    } finally {
      clearTimeout(timeout);
    }
    const text = await readLimited(response);
    if (!response.ok) {
      throw new MortiseError(
        `Mortise returned HTTP ${response.status}: ${text.trim()}`,
        response.status,
      );
    }
    let parsed: unknown;
    try {
      parsed = JSON.parse(text);
    } catch {
      throw new MortiseError("Mortise returned invalid JSON", response.status);
    }
    if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) {
      throw new MortiseError("Mortise response must be a JSON object", response.status);
    }
    return parsed as MortiseJSON;
  }
}

async function readLimited(response: Response): Promise<string> {
  const declaredLength = Number(response.headers.get("Content-Length"));
  if (Number.isFinite(declaredLength) && declaredLength > MAX_RESPONSE_BYTES) {
    throw new MortiseError("Mortise response exceeds 1 MiB", response.status);
  }
  if (!response.body) {
    return "";
  }
  const reader = response.body.getReader();
  const chunks: Uint8Array[] = [];
  let total = 0;
  while (true) {
    const { done, value } = await reader.read();
    if (done) {
      break;
    }
    total += value.byteLength;
    if (total > MAX_RESPONSE_BYTES) {
      await reader.cancel();
      throw new MortiseError("Mortise response exceeds 1 MiB", response.status);
    }
    chunks.push(value);
  }
  const data = new Uint8Array(total);
  let offset = 0;
  for (const chunk of chunks) {
    data.set(chunk, offset);
    offset += chunk.byteLength;
  }
  return new TextDecoder().decode(data);
}
