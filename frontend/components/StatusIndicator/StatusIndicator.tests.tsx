import React from "react";
import { render, screen, fireEvent } from "@testing-library/react";

import StatusIndicator from "./StatusIndicator";

describe("Status indicator", () => {
  it("renders status as capitalized", () => {
    render(<StatusIndicator value="paused" />);

    expect(screen.getByText("Paused")).toBeInTheDocument();
  });

  it("renders optional tooltip on hover", async () => {
    const TOOLTIP_TEXT = "Online hosts will respond to a live query.";
    render(
      <StatusIndicator value="online" tooltip={{ tooltipText: TOOLTIP_TEXT }} />
    );

    await fireEvent.mouseEnter(screen.getByText("Online"));

    expect(screen.getByText(TOOLTIP_TEXT)).toBeInTheDocument();
  });
});
