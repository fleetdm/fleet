import React from "react";
import { render, screen } from "@testing-library/react";
import ViewAllHostsLink from "./ViewAllHostsLink";

describe("ViewAllHostsLink - component", () => {
  it("renders View all hosts text and icon", () => {
    render(<ViewAllHostsLink />);

    const text = screen.getByText("View all hosts");
    const icon = screen.getByTestId("chevron-right-icon");

    expect(text).toBeInTheDocument();
    expect(icon).toBeInTheDocument();
  });

  it("hides text when set to condensed ", async () => {
    render(<ViewAllHostsLink queryParams={{ status: "online" }} condensed />);
    const text = screen.queryByText("View all hosts");

    expect(text).toBeNull();
  });
});
