import React from "react";

import { render, screen } from "@testing-library/react";

import { COLORS } from "styles/var/colors";

import ProgressBar from "./ProgressBar";

describe("ProgressBar component", () => {
  it("renders with the correct sections and colors", () => {
    const sections = [
      { color: "green", portion: 0.7 },
      { color: "red", portion: 0.1 },
    ];

    render(<ProgressBar sections={sections} />);

    const progressBar = screen.getByRole("progressbar");
    expect(progressBar).toBeInTheDocument();
    expect(progressBar).toHaveStyle({
      backgroundColor: COLORS["ui-fleet-black-10"],
    });

    const sectionElements = screen.getAllByTestId(/section-/);
    expect(sectionElements.length).toBe(2);

    // On CI, the rgb representation is used, while locally
    // it seems to use the named color.
    try {
      expect(sectionElements[0]).toHaveStyle(
        "background-color: rgb(0, 128, 0)"
      );
    } catch (error) {
      expect(sectionElements[0]).toHaveStyle("background-color: green");
    }

    expect(sectionElements[0]).toHaveStyle("width: 70%");

    // Check second section
    try {
      expect(sectionElements[1]).toHaveStyle(
        "background-color: rgb(255, 0, 0)"
      );
    } catch (error) {
      expect(sectionElements[1]).toHaveStyle("background-color: red");
    }
    expect(sectionElements[1]).toHaveStyle("width: 10%");
  });

  it("applies custom background color when provided", () => {
    const sections = [{ color: "green", portion: 0.5 }];
    const customBgColor = "blue";

    render(<ProgressBar sections={sections} backgroundColor={customBgColor} />);

    const progressBar = screen.getByRole("progressbar");
    try {
      expect(progressBar).toHaveStyle(`background-color: rgb(0, 0, 255)`);
    } catch (error) {
      expect(progressBar).toHaveStyle(`background-color: ${customBgColor}`);
    }
  });
});
