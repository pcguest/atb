import { describe, expect, it } from "vitest";

import {
  localViewerHref,
  localViewerNavigation,
  parseLocalViewerContext,
} from "@/lib/local-viewer-context";

const base =
  "/view/?bundle_path=%2Ftmp%2Fincident.atb&session_id=session-1&surface=evidence&seq=7&from=timeline&focus=atb%3A%2F%2Fevidence%2F1%2Fabc%3Fseq%3D7#session=secret-token";

describe("local viewer context", () => {
  it("parses bundle, session, surface, evidence and fragment state separately", () => {
    expect(parseLocalViewerContext(base)).toMatchObject({
      route: "/view",
      bundlePath: "/tmp/incident.atb",
      sessionId: "session-1",
      surface: "evidence",
      seq: 7,
      from: "timeline",
      focus: "atb://evidence/1/abc?seq=7",
      fragment: "#session=secret-token",
      discarded: [],
    });
  });

  it("accepts the legacy bundlePath alias and relative view routes", () => {
    expect(
      parseLocalViewerContext("/view?bundlePath=%2Ftmp%2Fold.atb&surface=run#session=t"),
    ).toMatchObject({
      route: "/view",
      bundlePath: "/tmp/old.atb",
      surface: "incident",
      fragment: "#session=t",
    });
  });

  it("does not confuse a focus locator with a bundle hash or session ID", () => {
    const hashLikeLocator = "a".repeat(64);
    const parsed = parseLocalViewerContext(
      `/view/?focus=${encodeURIComponent(hashLikeLocator)}#session=t`,
    );
    expect(parsed.focus).toBe(hashLikeLocator);
    expect(parsed.bundlePath).toBeNull();
    expect(parsed.sessionId).toBeNull();
  });

  it("discards malformed values without blocking valid content", () => {
    const parsed = parseLocalViewerContext(
      "/view/?bundle_path=%00bad&session_id=bad%0Avalue&surface=constructor&seq=notanumber&from=toString&focus=bad%00focus#session=t",
    );
    expect(parsed.route).toBe("/view");
    expect(parsed.bundlePath).toBeNull();
    expect(parsed.sessionId).toBeNull();
    expect(parsed.surface).toBeNull();
    expect(parsed.seq).toBeNull();
    expect(parsed.from).toBeNull();
    expect(parsed.focus).toBeNull();
    expect(parsed.fragment).toBe("#session=t");
    expect(parsed.discarded).toEqual(
      expect.arrayContaining(["bundle_path", "session_id", "surface", "seq", "from", "focus"]),
    );
  });

  it("ignores unknown query parameters", () => {
    const parsed = parseLocalViewerContext("/workspace?bundle_path=%2Ftmp%2Fa.atb&unknown=1#session=t");
    expect(parsed.route).toBe("/workspace");
    expect(parsed.bundlePath).toBe("/tmp/a.atb");
    expect(parsed.fragment).toBe("#session=t");
    expect(parsed.discarded).toEqual([]);
  });

  it("preserves all compatible view context and the fragment", () => {
    const parsed = parseLocalViewerContext(base);
    expect(localViewerHref("/view", parsed)).toBe(
      "/view/?bundle_path=%2Ftmp%2Fincident.atb&session_id=session-1&surface=evidence&seq=7&from=timeline&focus=atb%3A%2F%2Fevidence%2F1%2Fabc%3Fseq%3D7#session=secret-token",
    );
  });

  it("drops view-only context when moving to library routes and reports the discard", () => {
    const parsed = parseLocalViewerContext(base);
    const navigation = localViewerNavigation("/workspace", parsed);
    expect(navigation.href).toBe(
      "/workspace/?bundle_path=%2Ftmp%2Fincident.atb&session_id=session-1#session=secret-token",
    );
    expect(navigation.discarded).toEqual(["surface", "seq", "from", "focus"]);
  });

  it("keeps selected evidence when transitioning between compatible surfaces", () => {
    const parsed = parseLocalViewerContext(base);
    const navigation = localViewerNavigation("/view", parsed, {
      surface: "timeline",
      from: null,
      focus: null,
    });
    expect(navigation.href).toContain("surface=timeline");
    expect(navigation.href).toContain("seq=7");
    expect(navigation.href).not.toContain("from=");
    expect(navigation.href).not.toContain("focus=");
    expect(navigation.href.endsWith("#session=secret-token")).toBe(true);
  });

  it("writes public surface slugs rather than internal surface ids", () => {
    const parsed = parseLocalViewerContext("/view/#session=t");
    const href = localViewerHref("/view", parsed, { surface: "incident", from: "trust", seq: 2 });
    expect(href).toContain("surface=run");
    expect(href).toContain("from=status");
    expect(href).not.toContain("surface=incident");
    expect(href).not.toContain("from=trust");
  });

  it("preserves a source-return path for evidence", () => {
    const parsed = parseLocalViewerContext(base);
    const navigation = localViewerNavigation("/view", parsed, {
      surface: "evidence",
      seq: 7,
      from: "timeline",
    });
    expect(navigation.href).toContain("surface=evidence");
    expect(navigation.href).toContain("seq=7");
    expect(navigation.href).toContain("from=timeline");
  });

  it("encodes paths and locators without letting query text become a fragment", () => {
    const parsed = parseLocalViewerContext(
      `/view/?bundle_path=${encodeURIComponent("/tmp/a b#c&d.atb")}&focus=${encodeURIComponent("atb://evidence/1/x?seq=2&x=#frag")}#session=t`,
    );
    const href = localViewerHref("/view", parsed);
    expect(href).toContain("bundle_path=%2Ftmp%2Fa+b%23c%26d.atb");
    expect(href.split("#")[0]).not.toContain("session=t");
    expect(href.endsWith("#session=t")).toBe(true);
  });

  it("does not copy the session fragment into query parameters", () => {
    const parsed = parseLocalViewerContext(base);
    const href = localViewerHref("/sessions", parsed);
    const [path, fragment] = href.split("#");
    expect(path).not.toContain("session=secret-token");
    expect(path).not.toContain("secret-token");
    expect(fragment).toBe("session=secret-token");
  });

  it("fails safe for malformed direct navigation and reloads", () => {
    expect(parseLocalViewerContext("/view/?surface=bogus&seq=-1&from=toString")).toMatchObject({
      surface: null,
      seq: null,
      from: null,
      discarded: expect.arrayContaining(["surface", "seq", "from"]),
    });
    expect(parseLocalViewerContext("/not-a-view?surface=timeline&seq=2")).toMatchObject({
      route: null,
      surface: "timeline",
      seq: 2,
    });
  });

  it("is deterministic for back/forward-style reparsing", () => {
    const first = localViewerHref("/view", parseLocalViewerContext(base));
    const second = localViewerHref("/view", parseLocalViewerContext(first));
    expect(second).toBe(first);
  });
});
