import React from "react";

import { screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

import DiskSpaceIndicator from "./DiskSpaceIndicator";

describe("Disk space Indicator", () => {
  it("renders warning tooltip for <32gB when hovering over the yellow disk space indicator for darwin or windows", async () => {
    const { user } = renderWithSetup(
      <DiskSpaceIndicator
        baseClass="data-set"
        gigsDiskSpaceAvailable={17}
        percentDiskSpaceAvailable={10}
        id="disk-space-indicator"
        platform="darwin"
        tooltipPosition="bottom"
      />
    );

    expect(screen.getByTitle("disk space indicator")).toHaveStyle("width: 10%");
    expect(screen.getByTitle("disk space indicator")).toHaveClass(
      "data-set__disk-space--yellow"
    );

    await user.hover(screen.getByTitle("disk space indicator"));
    const tooltip = screen.getByText(
      "Not enough disk space available to install most large operating systems updates."
    );
    expect(tooltip).toBeInTheDocument();
  });

  it("renders severe warning tooltip for <16 gBwhen hovering over the red disk space indicator for darwin or windows", async () => {
    const { user } = renderWithSetup(
      <DiskSpaceIndicator
        baseClass="data-set"
        gigsDiskSpaceAvailable={5}
        percentDiskSpaceAvailable={2}
        id="disk-space-indicator"
        platform="windows"
        tooltipPosition="bottom"
      />
    );

    expect(screen.getByTitle("disk space indicator")).toHaveStyle("width: 2%");
    expect(screen.getByTitle("disk space indicator")).toHaveClass(
      "data-set__disk-space--red"
    );

    await user.hover(screen.getByTitle("disk space indicator"));
    const tooltip = screen.getByText(
      "Not enough disk space available to install most small operating systems updates."
    );
    expect(tooltip).toBeInTheDocument();
  });

  it("renders tooltip when hovering over the green disk space indicator for darwin or windows", async () => {
    const { user } = renderWithSetup(
      <DiskSpaceIndicator
        baseClass="data-set"
        gigsDiskSpaceAvailable={33}
        percentDiskSpaceAvailable={15}
        id="disk-space-indicator"
        platform="windows"
        tooltipPosition="bottom"
      />
    );

    expect(screen.getByTitle("disk space indicator")).toHaveStyle("width: 15%");
    expect(screen.getByTitle("disk space indicator")).toHaveClass(
      "data-set__disk-space--green"
    );

    await user.hover(screen.getByTitle("disk space indicator"));
    const tooltip = screen.getByText(
      "Enough disk space available to install most operating systems updates."
    );
    expect(tooltip).toBeInTheDocument();
  });
});
