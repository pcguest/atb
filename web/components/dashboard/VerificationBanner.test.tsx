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

  it("raises an explicit integrity alert on failure", () => {
    render(<VerificationBanner status="invalid" chainLength={7} headHash="sha256:abc123" bundlePath="/tmp/run.atb/bundle.atb" message="hash chain verification failed" />);
    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(screen.getByText("⚠ TAMPER DETECTED")).toBeInTheDocument();
  });
});
