import React from "react";
import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/testingUtils";

import RevealButton from "./RevealButton";

describe("Reveal button", () => {
  it("renders show text and hide text on click", async () => {
    const { user } = renderWithSetup(
      <RevealButton
        isShowing={false}
        hideText={"Hide advanced options"}
        showText={"Show advanced options"}
      />
    );

    await user.hover(screen.getByText("Show advanced options"));

    expect(screen.getByText(/to retrieve software/i)).toBeInTheDocument();
  });
  it("hides caret by default", async () => {
    render(
      <RevealButton
        isShowing={false}
        hideText={"Hide advanced options"}
        showText={"Show advanced options"}
        caretPosition={"before"}
      />
    );
  });
  it("renders caret on left", async () => {
    render(
      <RevealButton
        isShowing={false}
        hideText={"Hide advanced options"}
        showText={"Show advanced options"}
        caretPosition={"before"}
      />
    );
  });
  it("renders caret on right", async () => {
    render(
      <RevealButton
        isShowing={false}
        hideText={"Hide advanced options"}
        showText={"Show advanced options"}
        caretPosition={"before"}
      />
    );
  });
  it("renders tooltip on hover if provided", async () => {
    const { user } = renderWithSetup(
      <RevealButton
        isShowing={false}
        hideText={"Hide advanced options"}
        showText={"Show advanced options"}
        caretPosition={"before"}
      />
    );

    await user.hover(screen.getByText("Show advanced options"));

    expect(screen.getByText(/to retrieve software/i)).toBeInTheDocument();
  });
});
