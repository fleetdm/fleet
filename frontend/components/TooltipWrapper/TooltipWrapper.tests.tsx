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

  it("does not render tooltip when tipContent is empty", async () => {
    // Guarantees callers can pass a conditional/empty tipContent without
    // the tooltip's empty background flashing on hover.
    const { user } = renderWithSetup(
      <TooltipWrapper tipContent="">
        <span>Hover me</span>
      </TooltipWrapper>
    );

    const anchor = screen.getByText("Hover me");
    await user.hover(anchor);

    // The tooltip's root gets role="tooltip"; hovering an empty-content wrapper
    // must not mount one.
    await waitFor(() => {
      expect(screen.queryByRole("tooltip")).toBeNull();
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

  it("does not apply underline class when tipContent is empty", () => {
    render(
      <TooltipWrapper tipContent="">
        <span>Hover me</span>
      </TooltipWrapper>
    );
    const element = screen.getByText("Hover me").parentElement;
    expect(element).not.toHaveClass("component__tooltip-wrapper__underline");
  });

  it("does not apply underline class when disableTooltip is true", () => {
    render(
      <TooltipWrapper tipContent="Tooltip text" disableTooltip>
        <span>Hover me</span>
      </TooltipWrapper>
    );
    const element = screen.getByText("Hover me").parentElement;
    expect(element).not.toHaveClass("component__tooltip-wrapper__underline");
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

  it("wraps tipContent in a display:contents span by default (textBalanced)", async () => {
    const { user } = renderWithSetup(
      <TooltipWrapper tipContent="Balanced tooltip">
        <span>Hover me</span>
      </TooltipWrapper>
    );

    await user.hover(screen.getByText("Hover me"));

    await waitFor(() => {
      const tipText = screen.getByText("Balanced tooltip");
      // BalancedTipContent wraps content in a span with display:contents so
      // measurement can find the tooltip root via el.parentElement.
      const balancedWrapper = tipText.closest('span[style*="contents"]');
      expect(balancedWrapper).not.toBeNull();
    });
  });

  it("renders tipContent directly when textBalanced is false", async () => {
    const { user } = renderWithSetup(
      <TooltipWrapper tipContent="Unbalanced tooltip" textBalanced={false}>
        <span>Hover me</span>
      </TooltipWrapper>
    );

    await user.hover(screen.getByText("Hover me"));

    await waitFor(() => {
      const tipText = screen.getByText("Unbalanced tooltip");
      // Opt-out skips the BalancedTipContent span entirely — no display:contents
      // wrapper should exist anywhere in the tooltip's DOM.
      expect(tipText.closest('span[style*="contents"]')).toBeNull();
    });
  });

  it("does not throw when Range.getClientRects is unavailable (jsdom)", async () => {
    // jsdom doesn't implement Range.getClientRects; the effect should feature-
    // detect and no-op rather than throw. If the guard regresses this test
    // will surface as an unhandled TypeError during the hover.
    const errorSpy = jest
      .spyOn(console, "error")
      .mockImplementation(() => undefined);

    try {
      const { user } = renderWithSetup(
        <TooltipWrapper tipContent="Guarded tooltip">
          <span>Hover me</span>
        </TooltipWrapper>
      );

      await user.hover(screen.getByText("Hover me"));

      await waitFor(() => {
        expect(screen.getByText("Guarded tooltip")).toBeInTheDocument();
      });

      // BalancedTipContent's measurement runs inside a requestAnimationFrame
      // scheduled from useLayoutEffect — waitFor above may resolve before it
      // fires. Flush one animation frame so the getClientRects call (and any
      // TypeError it would throw without the guard) is captured by the spy
      // before we assert.
      await new Promise<void>((resolve) => {
        requestAnimationFrame(() => resolve());
      });

      // No TypeError from getClientRects should have been logged.
      const errorCalls = errorSpy.mock.calls.map((args) => String(args[0]));
      expect(
        errorCalls.some((msg) =>
          msg.includes("getClientRects is not a function")
        )
      ).toBe(false);
    } finally {
      errorSpy.mockRestore();
    }
  });
});
