import React from "react";

import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/testingUtils";

import BackLink from "./BackLink";

describe("BackLink - component", () => {
  it("renders text and icon", () => {
    render(<BackLink text="Back to software" />);

    const title = screen.getByText("Back to software");
    const icon = screen.queryByTitle("Icon");

    expect(title).toBeInTheDocument();
    expect(icon).toBeInTheDocument();
  });
  it("renders link on click", async () => {
    const { user } = renderWithSetup(<BackLink text="Back to software" />);

    await user.click(screen.getByText("Back to software"));

    // TODO: how to test a back link
    // expect(window.location.pathname).toBe(
    //   "https://github.com/fleetdm/fleet/issues/new/choose"
    // );
  });
});
