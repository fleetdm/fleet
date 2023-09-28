import React from "react";
import { screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

import PillCell from "./PillCell";

const PERFORMANCE_IMPACT = { indicator: "Minimal", id: 3 };

describe("Pill cell", () => {
  it("renders pill text and tooltip on hover", async () => {
    const { user } = renderWithSetup(<PillCell value={PERFORMANCE_IMPACT} />);

    await user.hover(screen.getByText("Minimal"));

    expect(screen.getByText(/little to no impact/i)).toBeInTheDocument();
  });
});
