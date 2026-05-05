import React from "react";
import { render, screen } from "@testing-library/react";
import CardHeader from "./CardHeader";

describe("CardHeader", () => {
  it("renders header text and subheader text when provided", () => {
    const headerText = "Test Header";
    const subheaderText = "Test Subheader";
    render(<CardHeader header={headerText} subheader={subheaderText} />);

    const header = screen.getByText(headerText);
    expect(header).toBeInTheDocument();
    expect(header.tagName).toBe("H2");
    const subheader = screen.getByText(subheaderText);
    expect(subheader).toBeInTheDocument();
    expect(subheader.tagName).toBe("P");
  });
  it("does not render subheader when not provided", () => {
    const headerText = "Test Header";
    render(<CardHeader header={headerText} />);

    const subheader = screen.queryByText(/subheader/i);
    expect(subheader).not.toBeInTheDocument();
  });
  it("renders JSX elements for header and subheader", () => {
    const headerJSX = <span data-testid="header-jsx">Header JSX</span>;
    const subheaderJSX = <span data-testid="subheader-jsx">Subheader JSX</span>;
    render(<CardHeader header={headerJSX} subheader={subheaderJSX} />);

    expect(screen.getByTestId("header-jsx")).toBeInTheDocument();
    expect(screen.getByTestId("subheader-jsx")).toBeInTheDocument();
  });
});
