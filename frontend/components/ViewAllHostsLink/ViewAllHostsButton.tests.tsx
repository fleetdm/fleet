import React from "react";
import { render, screen } from "@testing-library/react";
import ViewAllHostsButton from "./ViewAllHostsButton";

describe("ViewAllHostsButton - component", () => {
  it("renders View all hosts text and icon", () => {
    render(<ViewAllHostsButton />);

    const text = screen.getByText("View all hosts");
    const icon = screen.getByTestId("chevron-right-icon");

    expect(text).toBeInTheDocument();
    expect(icon).toBeInTheDocument();
  });

  it("hides text when set to condensed ", async () => {
    render(<ViewAllHostsButton queryParams={{ status: "online" }} condensed />);
    const text = screen.queryByText("View all hosts");

    expect(text).toBeNull();
  });
});
