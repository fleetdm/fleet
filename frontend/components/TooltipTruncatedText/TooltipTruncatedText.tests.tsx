import React from "react";
import { render } from "@testing-library/react";

import TooltipWrapper from "components/TooltipWrapper";
import { useCheckTruncatedElement } from "hooks/useCheckTruncatedElement";
import TooltipTruncatedText from "./TooltipTruncatedText";

// Mock TooltipWrapper so we can spy on the props TooltipTruncatedText forwards.
// We don't care about TooltipWrapper's internal behavior here — only that
// pass-through props (notably fixedPositionStrategy, disableTooltip, and
// tipContent) reach it correctly.
jest.mock("components/TooltipWrapper", () => ({
  __esModule: true,
  default: jest.fn(({ children }) => <div>{children}</div>),
}));

// Mock useCheckTruncatedElement so we can control isTruncated in tests.
// In jsdom there's no layout computation, so the real hook always returns
// false — mocking lets us assert the disableTooltip wiring in both states.
jest.mock("hooks/useCheckTruncatedElement", () => ({
  useCheckTruncatedElement: jest.fn(),
}));

const mockedTooltipWrapper = (TooltipWrapper as unknown) as jest.Mock;
const mockedUseCheckTruncatedElement = useCheckTruncatedElement as jest.Mock;

describe("TooltipTruncatedText", () => {
  beforeEach(() => {
    mockedTooltipWrapper.mockClear();
    mockedUseCheckTruncatedElement.mockReturnValue(false);
  });

  it("forwards fixedPositionStrategy=true to TooltipWrapper when set", () => {
    render(<TooltipTruncatedText value="example" fixedPositionStrategy />);

    expect(mockedTooltipWrapper.mock.calls[0][0]).toEqual(
      expect.objectContaining({ fixedPositionStrategy: true })
    );
  });

  it("defaults fixedPositionStrategy to false when not provided", () => {
    render(<TooltipTruncatedText value="example" />);

    expect(mockedTooltipWrapper.mock.calls[0][0]).toEqual(
      expect.objectContaining({ fixedPositionStrategy: false })
    );
  });

  it("disables the tooltip when text is not truncated", () => {
    mockedUseCheckTruncatedElement.mockReturnValue(false);

    render(<TooltipTruncatedText value="short" />);

    expect(mockedTooltipWrapper.mock.calls[0][0]).toEqual(
      expect.objectContaining({ disableTooltip: true })
    );
  });

  it("enables the tooltip when text is truncated", () => {
    mockedUseCheckTruncatedElement.mockReturnValue(true);

    render(
      <TooltipTruncatedText value="a very long value that gets truncated" />
    );

    expect(mockedTooltipWrapper.mock.calls[0][0]).toEqual(
      expect.objectContaining({ disableTooltip: false })
    );
  });

  it("uses value as the tip content when tooltip prop is not provided", () => {
    render(<TooltipTruncatedText value="just-the-value" />);

    expect(mockedTooltipWrapper.mock.calls[0][0]).toEqual(
      expect.objectContaining({ tipContent: "just-the-value" })
    );
  });

  it("uses tooltip prop as the tip content when provided, overriding value", () => {
    render(
      <TooltipTruncatedText
        value="display-value"
        tooltip="custom-tip-content"
      />
    );

    expect(mockedTooltipWrapper.mock.calls[0][0]).toEqual(
      expect.objectContaining({ tipContent: "custom-tip-content" })
    );
  });
});
