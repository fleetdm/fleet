import React from "react";
import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/testingUtils";

import PillCell from "./PillCell";

describe("Pill cell", () => {
  it("renders pill text", async () => {
    const { user } = renderWithSetup(
      <PillCell value={["ok", 3]} customIdPrefix={"4"} hostDetails />
    );

    await user.hover(screen.getByText("4"));

    expect(screen.getByText(/failing policies/i)).toBeInTheDocument();
  });

  it("renders tooltip on hover", async () => {
    const { user } = renderWithSetup(
      <PillCell value={["ok", 3]} customIdPrefix={"4"} hostDetails />
    );

    await user.hover(screen.getByText("Updated never"));

    expect(screen.getByText(/to retrieve software/i)).toBeInTheDocument();
  });
});
