import React from "react";
import { render, screen, waitFor } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

import StatusIndicator from "./StatusIndicator";

describe("Status indicator", () => {
  it("renders status as capitalized", () => {
    render(<StatusIndicator value="paused" />);

    expect(screen.getByText("Paused")).toBeInTheDocument();
  });

  it("renders optional tooltip on hover", async () => {
    const TOOLTIP_TEXT = "Online hosts will respond to a live query.";
    const { user } = renderWithSetup(
      <StatusIndicator value="online" tooltip={{ tooltipText: TOOLTIP_TEXT }} />
    );

    await user.hover(screen.getByText("Online"));

    await waitFor(() => {
      expect(screen.getByText(TOOLTIP_TEXT)).toBeInTheDocument();
    });
  });
});
