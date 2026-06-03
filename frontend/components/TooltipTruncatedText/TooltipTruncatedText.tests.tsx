import React from "react";
import { render } from "@testing-library/react";

import TooltipTruncatedText from "./TooltipTruncatedText";

// Mock TooltipWrapper so we can spy on the props TooltipTruncatedText forwards.
// We don't care about TooltipWrapper's internal behavior here — only that
// pass-through props (notably fixedPositionStrategy, disableTooltip, and
// tipContent) reach it correctly.
jest.mock("components/TooltipWrapper", () => {
  const mock = jest.fn(({ children }) => <div>{children}</div>);
  return { __esModule: true, default: mock };
});

// Mock useCheckTruncatedElement so we can control isTruncated in tests.
// In jsdom there's no layout computation, so the real hook always returns
// false — mocking lets us assert the disableTooltip wiring in both states.
jest.mock("hooks/useCheckTruncatedElement", () => ({
  useCheckTruncatedElement: jest.fn(),
}));

/* eslint-disable @typescript-eslint/no-var-requires */
const TooltipWrapper = require("components/TooltipWrapper").default as jest.Mock;
const { useCheckTruncatedElement } = require("hooks/useCheckTruncatedElement");
/* eslint-enable @typescript-eslint/no-var-requires */

describe("TooltipTruncatedText", () => {
  beforeEach(() => {
    TooltipWrapper.mockClear();
    (useCheckTruncatedElement as jest.Mock).mockReturnValue(false);
  });

  it("forwards fixedPositionStrategy=true to TooltipWrapper when set", () => {
    render(<TooltipTruncatedText value="example" fixedPositionStrategy />);

    expect(TooltipWrapper).toHaveBeenCalledWith(
      expect.objectContaining({ fixedPositionStrategy: true }),
      expect.anything()
    );
  });

  it("defaults fixedPositionStrategy to false when not provided", () => {
    render(<TooltipTruncatedText value="example" />);

    expect(TooltipWrapper).toHaveBeenCalledWith(
      expect.objectContaining({ fixedPositionStrategy: false }),
      expect.anything()
    );
  });

  it("disables the tooltip when text is not truncated", () => {
    (useCheckTruncatedElement as jest.Mock).mockReturnValue(false);

    render(<TooltipTruncatedText value="short" />);

    expect(TooltipWrapper).toHaveBeenCalledWith(
      expect.objectContaining({ disableTooltip: true }),
      expect.anything()
    );
  });

  it("enables the tooltip when text is truncated", () => {
    (useCheckTruncatedElement as jest.Mock).mockReturnValue(true);

    render(<TooltipTruncatedText value="a very long value that gets truncated" />);

    expect(TooltipWrapper).toHaveBeenCalledWith(
      expect.objectContaining({ disableTooltip: false }),
      expect.anything()
    );
  });

  it("uses value as the tip content when tooltip prop is not provided", () => {
    render(<TooltipTruncatedText value="just-the-value" />);

    expect(TooltipWrapper).toHaveBeenCalledWith(
      expect.objectContaining({ tipContent: "just-the-value" }),
      expect.anything()
    );
  });

  it("uses tooltip prop as the tip content when provided, overriding value", () => {
    render(
      <TooltipTruncatedText value="display-value" tooltip="custom-tip-content" />
    );

    expect(TooltipWrapper).toHaveBeenCalledWith(
      expect.objectContaining({ tipContent: "custom-tip-content" }),
      expect.anything()
    );
  });
});
