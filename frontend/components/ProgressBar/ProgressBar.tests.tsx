import React from "react";

import { render, screen } from "@testing-library/react";

import { COLORS } from "styles/var/colors";

import ProgressBar, { IProgressBarSection } from "./ProgressBar";

describe("ProgressBar component", () => {
  it("renders with the correct sections and colors", () => {
    const sections = [
      { color: "#5cb85c", portion: 0.7 },
      { color: "#d9534f", portion: 0.1 },
    ];

    render(<ProgressBar sections={sections} />);

    const progressBar = screen.getByRole("progressbar");
    expect(progressBar).toBeInTheDocument();
    expect(progressBar).toHaveStyle({
      backgroundColor: COLORS["ui-fleet-black-10"],
    });

    const sectionElements = screen.getAllByTestId(/section-/);
    expect(sectionElements.length).toBe(2);

    expect(sectionElements[0]).toHaveStyle({
      backgroundColor: "#5cb85c",
      width: "70%",
    });

    // Check second section
    expect(sectionElements[1]).toHaveStyle({
      backgroundColor: "#d9534f",
      width: "10%",
    });
  });

  it("applies custom background color when provided", () => {
    const sections = [{ color: "#5cb85c", portion: 0.5 }];
    const customBgColor = "red";

    render(<ProgressBar sections={sections} backgroundColor={customBgColor} />);

    const progressBar = screen.getByRole("progressbar");
    expect(progressBar).toHaveStyle({ backgroundColor: customBgColor });
  });
});
