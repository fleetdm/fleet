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
});
