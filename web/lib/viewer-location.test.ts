import { describe, expect, it } from "vitest";

import { clearViewerLocation, parseViewerLocation, viewerLocationUrl } from "./viewer-location";

describe("viewer location state", () => {
  it("parses a valid surface slug and sequence", () => {
    expect(parseViewerLocation("?surface=timeline&seq=5")).toEqual({
      surface: "timeline",
      seq: 5,
      from: null,
      unrecognisedSurface: null,
    });
  });

  it("maps legible slugs to internal surface ids", () => {
    expect(parseViewerLocation("?surface=run").surface).toBe("incident");
    expect(parseViewerLocation("?surface=status").surface).toBe("trust");
  });

  it("reports an unknown or internal-id surface instead of silently ignoring it", () => {
    expect(parseViewerLocation("?surface=bogus")).toEqual({
      surface: null,
      seq: null,
      from: null,
      unrecognisedSurface: "bogus",
    });
    expect(parseViewerLocation("?surface=trust")).toEqual({
      surface: null,
      seq: null,
      from: null,
      unrecognisedSurface: "trust",
    });
  });

  it("treats prototype keys and empty surface values as unrecognised (no prototype access)", () => {
    expect(parseViewerLocation("?surface=constructor")).toEqual({
      surface: null,
      seq: null,
      from: null,
      unrecognisedSurface: "constructor",
    });
    expect(parseViewerLocation("?surface=toString&seq=3").seq).toBeNull();
    expect(parseViewerLocation("?surface=")).toEqual({
      surface: null,
      seq: null,
      from: null,
      unrecognisedSurface: "(empty)",
    });
    expect(parseViewerLocation("?surface=evidence&from=constructor").from).toBeNull();
  });

  it("does not accept a sequence without a valid surface", () => {
    expect(parseViewerLocation("?surface=bogus&seq=5").seq).toBeNull();
    expect(parseViewerLocation("?seq=5").seq).toBeNull();
  });

  it("returns null seq for a non-numeric, negative or fractional sequence", () => {
    expect(parseViewerLocation("?surface=evidence&seq=abc")).toEqual({
      surface: "evidence",
      seq: null,
      from: null,
      unrecognisedSurface: null,
    });
    expect(parseViewerLocation("?surface=evidence&seq=-1").seq).toBeNull();
    expect(parseViewerLocation("?surface=evidence&seq=1.5").seq).toBeNull();
  });

  it("parses a return-path origin, ignoring an invalid one", () => {
    expect(parseViewerLocation("?surface=evidence&seq=7&from=timeline").from).toBe("timeline");
    expect(parseViewerLocation("?surface=evidence&seq=7&from=bogus").from).toBeNull();
  });

  it("returns nulls when no presentation location is present", () => {
    expect(parseViewerLocation("")).toEqual({ surface: null, seq: null, from: null, unrecognisedSurface: null });
  });

  it("writes a legible surface slug and sequence without disturbing the session fragment", () => {
    expect(viewerLocationUrl("http://127.0.0.1:18888/view/#session=abc", "evidence", 7))
      .toBe("/view/?surface=evidence&seq=7#session=abc");
    expect(viewerLocationUrl("http://127.0.0.1:18888/view/", "incident", null))
      .toBe("/view/?surface=run");
  });

  it("records a return-path origin, but never a self-return", () => {
    expect(viewerLocationUrl("http://127.0.0.1:18888/view/", "evidence", 7, "timeline"))
      .toBe("/view/?surface=evidence&seq=7&from=timeline");
    expect(viewerLocationUrl("http://127.0.0.1:18888/view/", "evidence", 7, "evidence"))
      .toBe("/view/?surface=evidence&seq=7");
  });

  it("removes the sequence when there is none and drops a resolved focus reference", () => {
    const url = viewerLocationUrl("http://127.0.0.1:18888/view/?focus=atb%3A%2F%2Fevidence&seq=3", "incident", null);
    expect(url).not.toContain("seq=");
    expect(url).not.toContain("focus=");
    expect(url).toContain("surface=run");
  });
});

describe("clearViewerLocation", () => {
  it("drops presentation state but preserves the evidence locator and the session fragment", () => {
    const cleared = clearViewerLocation(
      "http://127.0.0.1:18888/view/?surface=evidence&seq=7&from=timeline&focus=atb%3A%2F%2Fevidence#session=secret-token",
    );
    expect(cleared).not.toContain("surface=");
    expect(cleared).not.toContain("seq=");
    expect(cleared).not.toContain("from=");
    expect(cleared).toContain("focus=atb%3A%2F%2Fevidence");
    expect(cleared).toContain("#session=secret-token");
  });
});
