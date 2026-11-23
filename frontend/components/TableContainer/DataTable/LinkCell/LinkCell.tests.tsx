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

  it("renders the truncation tooltip outside the link so clicking it does not follow the link", async () => {
    const onClick = jest.fn();
    const { container, user } = renderWithSetup(
      <LinkCell
        value={VALUE}
        path={PATHS.MANAGE_HOSTS}
        customOnClick={onClick}
        tooltipTruncate
      />
    );

    // The tooltip overlaps the row above, so a click on it must not navigate to this row.
    const link = container.querySelector("a");
    const tooltip = document.querySelector(".__react_component_tooltip");
    expect(tooltip).toBeInTheDocument();
    expect(link).not.toContainElement(tooltip as HTMLElement);

    await user.click(tooltip as HTMLElement);
    expect(onClick).not.toHaveBeenCalled();
  });
});
