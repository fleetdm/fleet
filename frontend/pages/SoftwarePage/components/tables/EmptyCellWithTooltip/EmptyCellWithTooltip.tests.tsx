import { render, screen, waitFor } from "@testing-library/react";
import React from "react";

import { renderWithSetup } from "test/test-utils";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

import EmptyCellWithTooltip from "./EmptyCellWithTooltip";

describe("EmptyCellWithTooltip", () => {
  it("shows the tooltip on hover", async () => {
    const { user } = renderWithSetup(
      <EmptyCellWithTooltip tipContent="No versions here." />
    );

    await user.hover(screen.getByText(DEFAULT_EMPTY_CELL_VALUE));

    await waitFor(() => {
      expect(screen.getByText("No versions here.")).toBeInTheDocument();
    });
  });

  it("renders a plain empty value without tooltip content", () => {
    render(<EmptyCellWithTooltip />);

    expect(screen.getByText(DEFAULT_EMPTY_CELL_VALUE)).toBeInTheDocument();
    expect(screen.queryByRole("tooltip")).not.toBeInTheDocument();
  });
});
