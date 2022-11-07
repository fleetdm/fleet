import React from "react";

import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/testingUtils";

import ViewAllHostsLink from "./ViewAllHostsLink";

describe("CustomLink - component", () => {
  it("View all hosts link renders text and icon", () => {
    render(<ViewAllHostsLink />);

    const title = screen.getByText("View all hosts");
    const icon = screen.queryByTitle("Icon");

    expect(title).toBeInTheDocument();
    expect(icon).toBeInTheDocument();
  });

  it("renders link on click", async () => {
    const { user } = renderWithSetup(
      <ViewAllHostsLink queryParams={{ status: "online" }} />
    );

    await user.click(screen.getByText("View all hosts"));

    // TODO: how to test a link
    expect(window.location.pathname).toBe("/hosts/manage/&status=online");
  });
});
