import React from "react";
import { screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";
import PATHS from "router/paths";

import LinkCell from "./LinkCell";

const VALUE = "40 hosts";
describe("Link cell", () => {
  it("renders text and path", async () => {
    const { user } = renderWithSetup(
      <LinkCell value={VALUE} path={PATHS.MANAGE_HOSTS} />
    );

    await user.click(screen.getByText("40 hosts"));

    expect(window.location.pathname).toContain("/hosts");
  });
});
