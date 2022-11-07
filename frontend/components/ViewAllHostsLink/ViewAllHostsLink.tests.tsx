import React from "react";

import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/testingUtils";

import ViewAllHostsLink from "./ViewAllHostsLink";

describe("ViewAllHostsLink - component", () => {
  it("renders View all hosts text and icon", () => {
    render(<ViewAllHostsLink />);

    const text = screen.getByText("View all hosts");
    const icon = screen.getByTestId("Icon");

    expect(text).toBeInTheDocument();
    expect(icon).toBeInTheDocument();
  });

  it("renders link", () => {
    render(<ViewAllHostsLink queryParams={{ status: "online" }} />);

    const text = screen.queryByText("View all hosts");

    if (!text) {
      throw new Error("View all host text is null");
    }
    // console.log("text.closest(a)", text.closest("a"));
    const link = screen.getByRole("link", { name: "View all hosts" });
    // TODO: How to test partial link
    expect(link).toHaveAttribute("href", "/hosts/manage/&status=online");

    console.log("link", link);
  });

  it("hides text when set to condensed ", async () => {
    render(<ViewAllHostsLink queryParams={{ status: "online" }} condensed />);
    const text = screen.queryByText("View all hosts");

    expect(text).toBeNull();
  });
});
