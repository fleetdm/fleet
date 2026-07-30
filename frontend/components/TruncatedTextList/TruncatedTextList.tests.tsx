import React from "react";
import { render } from "@testing-library/react";

import TooltipWrapper from "components/TooltipWrapper";
import TruncatedTextList from "./TruncatedTextList";

// Mock TooltipWrapper so we can spy on which pieces of the row get wrapped
// (and with what tipContent) without pulling in the real tooltip's layout
// side effects.
jest.mock("components/TooltipWrapper", () => ({
  __esModule: true,
  default: jest.fn(({ children }) => <>{children}</>),
}));

const mockedTooltipWrapper = (TooltipWrapper as unknown) as jest.Mock;

// The component renders a hidden `__measure` layer that always contains
// "+{items.length} more" for width probing — target only the visible row so
// the measurement text can't false-positive our assertions.
const getVisibleRow = (container: HTMLElement) =>
  container.querySelector(".truncated-text-list__visible");

// In jsdom `getBoundingClientRect().width` is 0 for every element, so the
// internal measurement pass always concludes "the first item doesn't fit
// alongside a +N more pill" and falls into the `truncatedFirstContent`
// branch (visibleCount === 0). Both regressions guarded here live in that
// branch, which makes it the right one to pin down.
describe("TruncatedTextList — truncatedFirstContent edge cases", () => {
  beforeEach(() => {
    mockedTooltipWrapper.mockClear();
  });

  it("suppresses the '+N more' pill when items.length === 1", () => {
    const { container } = render(<TruncatedTextList items={["Solo"]} />);

    // Guards the "+0 more" regression: with a single item there is nothing
    // to hide, so the visible row must not render the pill at all.
    expect(getVisibleRow(container)).not.toHaveTextContent(/\+\d+ more/);
    expect(mockedTooltipWrapper).not.toHaveBeenCalled();
  });

  it("still renders the '+N more' pill when items.length > 1", () => {
    const { container } = render(
      <TruncatedTextList items={["Solo", "Duet"]} />
    );

    expect(getVisibleRow(container)).toHaveTextContent("+1 more");
  });

  it("does not wrap the first item in a tooltip when truncateString leaves it unchanged", () => {
    // 4 chars is well under the 30-char default truncatedFirstMaxChars, so
    // truncateString returns the value verbatim → no hover tooltip needed.
    render(<TruncatedTextList items={["Solo"]} />);

    expect(mockedTooltipWrapper).not.toHaveBeenCalled();
  });

  it("wraps the first item in a tooltip only when truncateString actually shortened it", () => {
    const longName =
      "A very long name that definitely exceeds thirty characters";
    render(<TruncatedTextList items={[longName]} />);

    expect(mockedTooltipWrapper).toHaveBeenCalled();
    expect(mockedTooltipWrapper.mock.calls[0][0]).toEqual(
      expect.objectContaining({ tipContent: longName })
    );
  });
});
