import { screen } from "@testing-library/react";
import React from "react";

import PATHS from "router/paths";
import { renderWithSetup } from "test/test-utils";

import LinkCell from "./LinkCell";

const VALUE = "40 hosts";
describe("Link cell", () => {
  it("renders text", async () => {
    renderWithSetup(<LinkCell value={VALUE} path={PATHS.MANAGE_HOSTS} />);

    expect(screen.getByText("40 hosts")).toBeInTheDocument();
    // Note: Testing react-router Link would require Router or MemoryRouter wrapper which is app level
  });

  it("renders a truncation tooltip that does not capture clicks meant for the row above", () => {
    const { container } = renderWithSetup(
      <LinkCell value={VALUE} path={PATHS.MANAGE_HOSTS} tooltipTruncate />
    );

    // react-tooltip's allow_click class switches the tooltip to pointer-events: auto, which lets it sit on top of
    // the row above and navigate to this row's link when clicked.
    const tooltip = container.querySelector(".__react_component_tooltip");
    expect(tooltip).toBeInTheDocument();
    expect(tooltip).not.toHaveClass("allow_click");
  });
});
