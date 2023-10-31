import React from "react";
import { render, screen } from "@testing-library/react";
import BackLink from "./BackLink";

describe("BackLink - component", () => {
  it("renders text and icon", () => {
    render(<BackLink text="Back to software" />);

    const text = screen.getByText("Back to software");
    const icon = screen.getByTestId("chevron-left-icon");

    expect(text).toBeInTheDocument();
    expect(icon).toBeInTheDocument();
  });
});
