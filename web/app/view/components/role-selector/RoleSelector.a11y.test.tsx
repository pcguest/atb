import React from "react";
import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { RoleSelector } from "@/app/view/components/role-selector/RoleSelector";

describe("RoleSelector accessibility", () => {
  it("exposes an accessible role selector control", () => {
    const { getByLabelText } = render(<RoleSelector />);

    const selector = getByLabelText("Select dashboard role");
    expect(selector).toBeInTheDocument();
  });
});
