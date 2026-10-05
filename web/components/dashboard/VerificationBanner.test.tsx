import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { VerificationBanner } from "./VerificationBanner";

describe("VerificationBanner", () => {
  it("leads with an integrity summary and keeps exact chain/hash detail behind a disclosure", () => {
    render(<VerificationBanner status="valid" chainLength={7} headHash="sha256:abc123" bundlePath="/tmp/run.atb/bundle.atb" />);
    expect(screen.getByText("Hash chain verified")).toBeInTheDocument();
    expect(screen.getByText(/7 recorded events/)).toBeInTheDocument();
    // The raw head hash is not at L1 orientation.
    expect(screen.queryByText("sha256:abc123")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Integrity details" }));
    expect(screen.getByText("sha256:abc123")).toBeVisible();
    expect(screen.getByText(/does not establish truth, completeness, or independent custody/)).toBeVisible();
  });

  it("does not present a trust, confidence or completeness score", () => {
    render(<VerificationBanner status="valid" chainLength={7} headHash="sha256:abc123" />);
    expect(screen.queryByText(/trust score|confidence|% complete/i)).not.toBeInTheDocument();
  });

  it("raises an explicit integrity alert without asserting tampering", () => {
    render(
      <VerificationBanner
        status="invalid"
        chainLength={7}
        headHash="sha256:abc123"
        bundlePath="/tmp/run.atb/bundle.atb"
        message="hash: verify: tamper detected at event 3 (seq 3): expected a, got b"
      />,
    );
    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(screen.getByText("⚠ INTEGRITY CHECK FAILED")).toBeInTheDocument();
    // ATB's own voice must bound the conclusion, and the exact head hash is not
    // exposed at L1.
    expect(screen.getByText(/recorded hash-chain verification did not succeed/i)).toBeInTheDocument();
    expect(screen.queryByText(/tamper/i)).not.toBeInTheDocument();
    expect(screen.queryByText("sha256:abc123")).not.toBeInTheDocument();
  });

  it("keeps the raw verifier output and head hash behind the disclosure", () => {
    render(
      <VerificationBanner
        status="invalid"
        chainLength={7}
        headHash="sha256:abc123"
        bundlePath="/tmp/run.atb/bundle.atb"
        message="hash: verify: tamper detected at event 3 (seq 3): expected a, got b"
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: "Integrity details" }));
    expect(screen.getByText("sha256:abc123")).toBeVisible();
    // Bounded conclusion: it must not assert that the recorded order is wrong.
    expect(screen.getByText(/could not be verified against their recorded hash chain and order/i)).toBeVisible();
    expect(screen.getByText(/Verifier output \(verbatim\)/)).toBeVisible();
    // The supplied verifier output itself must be rendered, not only its label.
    expect(screen.getByText(/tamper detected at event 3 \(seq 3\): expected a, got b/)).toBeVisible();
  });

  it("does not reference an unmounted panel while the disclosure is closed", () => {
    render(<VerificationBanner status="valid" chainLength={7} headHash="sha256:abc123" />);
    const toggle = screen.getByRole("button", { name: "Integrity details" });
    expect(toggle).toHaveAttribute("aria-expanded", "false");
    expect(toggle).not.toHaveAttribute("aria-controls");
    fireEvent.click(toggle);
    expect(toggle).toHaveAttribute("aria-expanded", "true");
    expect(toggle).toHaveAttribute("aria-controls");
    expect(screen.getByText("sha256:abc123")).toBeVisible();
  });
});
