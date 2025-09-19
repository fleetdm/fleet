import React from "react";

import { screen, waitFor } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

import { COLORS } from "styles/var/colors";

import DiskSpaceIndicator from "./DiskSpaceIndicator";

describe("Disk space Indicator", () => {
  const [errStyle, warningStyle, okayStyle] = [
    `background-color: ${COLORS["ui-error"]}`,
    `background-color: ${COLORS["ui-warning"]}`,
    `background-color: ${COLORS["status-success"]}`,
  ];
  it("renders warning tooltip for <32gB when hovering over the yellow disk space indicator for darwin or windows", async () => {
    const { user } = renderWithSetup(
      <DiskSpaceIndicator
        gigsDiskSpaceAvailable={17}
        percentDiskSpaceAvailable={10}
        platform="darwin"
        barTooltipPosition="bottom"
      />
    );

    const section = screen.getByTestId("section-0");
    expect(section).toHaveStyle(warningStyle);

    await user.hover(section);
    await waitFor(() => {
      const tooltip = screen.getByText(
        "Not enough disk space available to install most large operating systems updates."
      );
      expect(tooltip).toBeInTheDocument();
    });
  });

  it("renders severe warning tooltip for <16 gBwhen hovering over the red disk space indicator for darwin or windows", async () => {
    const { user } = renderWithSetup(
      <DiskSpaceIndicator
        gigsDiskSpaceAvailable={5}
        percentDiskSpaceAvailable={2}
        platform="windows"
        barTooltipPosition="bottom"
      />
    );

    const section = screen.getByTestId("section-0");
    expect(section).toHaveStyle(errStyle);

    await user.hover(section);

    await waitFor(() => {
      const tooltip = screen.getByText(
        "Not enough disk space available to install most small operating systems updates."
      );
      expect(tooltip).toBeInTheDocument();
    });
  });

  it("renders tooltip when hovering over the green disk space indicator for darwin or windows", async () => {
    const { user } = renderWithSetup(
      <DiskSpaceIndicator
        gigsDiskSpaceAvailable={33}
        percentDiskSpaceAvailable={15}
        platform="windows"
        barTooltipPosition="bottom"
      />
    );

    const section = screen.getByTestId("section-0");
    expect(section).toHaveStyle(okayStyle);

    await user.hover(section);

    await waitFor(() => {
      const tooltip = screen.getByText(
        "Enough disk space available to install most operating systems updates."
      );
      expect(tooltip).toBeInTheDocument();
    });
  });
});
