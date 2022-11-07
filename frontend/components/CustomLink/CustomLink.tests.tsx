import React from "react";

import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/testingUtils";

import CustomLink from "./CustomLink";

describe("CustomLink - component", () => {
  it("renders text and no icon", () => {
    render(
      <CustomLink
        url="https://github.com/fleetdm/fleet/issues/new/choose"
        text="file an issue"
      />
    );

    const title = screen.getByText("file an issue");
    const icon = screen.queryByTitle("Icon");

    expect(title).toBeInTheDocument();
    expect(icon).toBeNull();
  });

  it("renders icon if newTab is set", () => {
    render(
      <CustomLink
        url="https://github.com/fleetdm/fleet/issues/new/choose"
        text="file an issue"
        newTab
      />
    );

    const icon = screen.getByTitle("Icon");

    expect(icon).toBeInTheDocument();
  });

  it("renders link on click", async () => {
    const { user } = renderWithSetup(
      <CustomLink
        url="https://github.com/fleetdm/fleet/issues/new/choose"
        text="file an issue"
      />
    );

    await user.click(screen.getByText("file an issue"));

    // TODO: how to test a link
    // expect(window.location.pathname).toBe(
    //   "https://github.com/fleetdm/fleet/issues/new/choose"
    // );
  });
});
