import { describe, expect, it, vi } from "vitest";

import { MortiseClient, MortiseError } from "./mortise-client.js";

describe("MortiseClient", () => {
  it("runs authenticated custody, verification, and reverse lookup flows", async () => {
    const seenAuthorization: Array<string | null> = [];
    const fetchMock = vi.fn(async (input: string | URL | Request, init?: RequestInit) => {
      seenAuthorization.push(new Headers(init?.headers).get("Authorization"));
      const url = String(input);
      const path = new URL(url).pathname;
      const query = new URL(url).search;
      if (path === "/ingest") {
        return Response.json(
          {
            receipt_version: "custos.receipt.v1",
            receipt_id: "receipt-1",
            bundle_hash: "bundle-hash",
            attestation: { algorithm: "ed25519" },
          },
          { status: 201 },
        );
      }
      if (path === "/verify/bundle") {
        return Response.json({ verified: true, bundle_hash: "bundle-hash" });
      }
      if (path === "/verify/receipt") {
        return Response.json({ verified: true, receipt_id: "receipt-1" });
      }
      if (path === "/receipts/by-hash" && query === "?bundle_hash=bundle-hash") {
        return Response.json({ bundle_hash: "bundle-hash", count: 1, receipts: [] });
      }
      return Response.json({ error: "not found" }, { status: 404 });
    });
    const client = new MortiseClient("https://mortise.example/", {
      token: "secret",
      fetch: fetchMock,
    });

    const receipt = await client.ingestBundle(new TextEncoder().encode("bundle"));
    expect(receipt.receipt_id).toBe("receipt-1");
    expect(receipt.attestation).toEqual({ algorithm: "ed25519" });
    expect(await client.verifyBundle(new TextEncoder().encode("bundle"))).toMatchObject({
      verified: true,
    });
    expect(await client.verifyReceipt(receipt)).toEqual({
      verified: true,
      receipt_id: "receipt-1",
    });
    expect(await client.receiptsByHash("bundle-hash")).toMatchObject({ count: 1 });

    expect(seenAuthorization).toHaveLength(4);
    expect(seenAuthorization.every((value) => value === "Bearer secret")).toBe(true);
  });

  it.each([
    "",
    "mortise.example",
    "ftp://mortise.example",
    "https://user:pass@mortise.example",
    "https://mortise.example?token=secret",
    "https://mortise.example/#fragment",
  ])("rejects unsafe endpoint %s", (endpoint) => {
    expect(() => new MortiseClient(endpoint)).toThrow("invalid Mortise endpoint");
  });

  it("surfaces HTTP errors without exposing unbounded response bodies", async () => {
    const client = new MortiseClient("https://mortise.example", {
      fetch: vi.fn(async () => new Response("invalid bundle", { status: 422 })),
    });
    await expect(client.ingestBundle(new Uint8Array([1]))).rejects.toBeInstanceOf(
      MortiseError,
    );
    await expect(client.ingestBundle(new Uint8Array([1]))).rejects.toThrow("422");
  });

  it("rejects an invalid custody receipt", async () => {
    const client = new MortiseClient("https://mortise.example", {
      fetch: vi.fn(async () => Response.json({ receipt_version: "unknown" })),
    });
    await expect(client.ingestBundle(new Uint8Array([1]))).rejects.toThrow(
      "invalid custody receipt",
    );
  });
});
