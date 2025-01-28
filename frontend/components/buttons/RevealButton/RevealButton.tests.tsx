import React from "react";
import { render, screen, fireEvent } from "@testing-library/react";

import RevealButton from "./RevealButton";

const SHOW_TEXT = "Show advanced options";
const HIDE_TEXT = "Hide advanced options";
const TOOLTIP_CONTENT = "Customize logging type and platforms";

describe("Reveal button", () => {
  it("renders show text", async () => {
    render(
      <RevealButton
        isShowing={false}
        hideText={HIDE_TEXT}
        showText={SHOW_TEXT}
      />
    );

    const showText = screen.getByText(SHOW_TEXT);
    expect(showText).toBeInTheDocument();
  });

  it("renders hide text", async () => {
    render(
      <RevealButton isShowing hideText={HIDE_TEXT} showText={SHOW_TEXT} />
    );

    const hideText = screen.getByText(HIDE_TEXT);
    expect(hideText).toBeInTheDocument();
  });

  it("hides caret by default", async () => {
    render(
      <RevealButton
        isShowing={false}
        hideText={HIDE_TEXT}
        showText={SHOW_TEXT}
      />
    );

    const icon = screen.queryByTestId("chevron-down-icon");

    expect(icon).toBeNull();
  });

  it("renders caret on left", async () => {
    render(
      <RevealButton
        isShowing={false}
        hideText={HIDE_TEXT}
        showText={SHOW_TEXT}
        caretPosition="before"
      />
    );

    const icon = screen.queryByTestId("chevron-right-icon");
    expect(icon?.nextSibling).toHaveTextContent(SHOW_TEXT);
  });

  it("renders caret on right", async () => {
    render(
      <RevealButton
        isShowing={false}
        hideText={HIDE_TEXT}
        showText={SHOW_TEXT}
        caretPosition="after"
      />
    );

    const icon = screen.queryByTestId("chevron-down-icon");

    expect(icon?.previousSibling).toHaveTextContent(SHOW_TEXT);
  });

  it("renders tooltip on hover if provided", async () => {
    render(
      <RevealButton
        isShowing={false}
        hideText={HIDE_TEXT}
        showText={SHOW_TEXT}
        caretPosition="before"
        tooltipContent={TOOLTIP_CONTENT}
      />
    );

    await fireEvent.mouseEnter(screen.getByText(SHOW_TEXT));

    expect(screen.getByText(TOOLTIP_CONTENT)).toBeInTheDocument();
  });
});
