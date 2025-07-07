// TooltipWrapper.test.tsx
import React from "react";
import userEvent from "@testing-library/user-event";
import { render, screen } from "@testing-library/react";
import TooltipWrapper from "./TooltipWrapper";

describe("TooltipWrapper", () => {
  it("renders children and tooltip content", async () => {
    render(
      <TooltipWrapper tipContent="Tooltip text">
        <span>Hover me</span>
      </TooltipWrapper>
    );

    const trigger = screen.getByText("Hover me");
    userEvent.hover(trigger);

    // Wait for tooltip content to appear in the DOM
    expect(await screen.findByText("Tooltip text")).toBeInTheDocument();
  });

  it("does not render tooltip when disableTooltip is true", () => {
    render(
      <TooltipWrapper tipContent="Tooltip text" disableTooltip>
        <span>Hover me</span>
      </TooltipWrapper>
    );
    expect(screen.getByText("Hover me")).toBeInTheDocument();
    // Tooltip content should not be in the DOM
    expect(screen.queryByText("Tooltip text")).toBeNull();
  });

  it("applies underline class by default", () => {
    render(
      <TooltipWrapper tipContent="Tooltip text">
        <span>Hover me</span>
      </TooltipWrapper>
    );
    const element = screen.getByText("Hover me").parentElement;
    expect(element).toHaveClass("component__tooltip-wrapper__element");
    expect(element).toHaveClass("component__tooltip-wrapper__underline");
  });

  it("does not apply underline class when underline is false", () => {
    render(
      <TooltipWrapper tipContent="Tooltip text" underline={false}>
        <span>Hover me</span>
      </TooltipWrapper>
    );
    const element = screen.getByText("Hover me").parentElement;
    expect(element).not.toHaveClass("component__tooltip-wrapper__underline");
  });
});
