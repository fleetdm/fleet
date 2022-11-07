import React from "react";

import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/testingUtils";

import ViewAllHostsLink from "./ViewAllHostsLink";

describe("ViewAllHostsLink - component", () => {
  it("renders View all hosts text and icon", () => {
    render(<ViewAllHostsLink />);

    const title = screen.getByText("View all hosts");
    const icon = screen.queryByTitle("Icon");

    expect(title).toBeInTheDocument();
    expect(icon).toBeInTheDocument();
  });

  it("renders link", () => {
    render(<ViewAllHostsLink queryParams={{ status: "online" }} />);

    const text = screen.queryByText("View all hosts");

    if (!text) {
      throw new Error("View all host text is null");
    }

    // TODO: How to test partial link
    // expect(text.closest("a")).toHaveAttribute(
    //   "href",
    //   "/hosts/manage/&status=online"
    // );
  });

  it("hides text when set to condensed ", async () => {
    render(<ViewAllHostsLink queryParams={{ status: "online" }} condensed />);
    const title = screen.queryByText("View all hosts");

    expect(title).toBeNull();
  });
});
