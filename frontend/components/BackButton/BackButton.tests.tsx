import React from "react";
import { render, screen } from "@testing-library/react";
import BackButton from "./BackButton";

describe("BackButton - component", () => {
  it("renders text and icon", () => {
    render(<BackButton text="Back to software" />);

    const text = screen.getByText("Back to software");
    const icon = screen.getByTestId("chevron-left-icon");

    expect(text).toBeInTheDocument();
    expect(icon).toBeInTheDocument();
  });
});
