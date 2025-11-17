// TooltipWrapper.test.tsx
import React from "react";
import { render, screen, waitFor } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";
import TooltipWrapper from "./TooltipWrapper";

describe("TooltipWrapper", () => {
  it("renders children and tooltip content", async () => {
    const { user } = renderWithSetup(
      <TooltipWrapper tipContent="Tooltip text">
        <span>Hover me</span>
      </TooltipWrapper>
    );

    const trigger = screen.getByText("Hover me");
    await user.hover(trigger);

    await waitFor(() => {
      expect(screen.getByText("Tooltip text")).toBeInTheDocument();
    });
  });

  it("does not render tooltip when disableTooltip is true", async () => {
    const { user } = renderWithSetup(
      <TooltipWrapper tipContent="Tooltip text" disableTooltip>
        <span>Hover me</span>
      </TooltipWrapper>
    );
    const anchor = screen.getByText("Hover me");
    expect(anchor).toBeInTheDocument();

    await user.hover(anchor);

    await waitFor(() => {
      expect(screen.queryByText("Tooltip text")).toBeNull();
    });
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
