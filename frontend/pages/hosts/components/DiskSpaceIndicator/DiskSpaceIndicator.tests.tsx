import React from "react";

import { screen, render, fireEvent } from "@testing-library/react";

import DiskSpaceIndicator from "./DiskSpaceIndicator";

describe("Disk space Indicator", () => {
  it("renders warning tooltip for <32gB when hovering over the yellow disk space indicator for darwin or windows", async () => {
    render(
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

    await fireEvent.mouseOver(screen.getByTitle("disk space indicator"));
    const tooltip = screen.getByText(
      "Not enough disk space available to install most large operating systems updates."
    );
    expect(tooltip).toBeInTheDocument();
  });

  it("renders severe warning tooltip for <16 gBwhen hovering over the red disk space indicator for darwin or windows", async () => {
    render(
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

    await fireEvent.mouseOver(screen.getByTitle("disk space indicator"));
    const tooltip = screen.getByText(
      "Not enough disk space available to install most small operating systems updates."
    );
    expect(tooltip).toBeInTheDocument();
  });

  it("renders tooltip when hovering over the green disk space indicator for darwin or windows", async () => {
    render(
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

    await fireEvent.mouseOver(screen.getByTitle("disk space indicator"));
    const tooltip = screen.getByText(
      "Enough disk space available to install most operating systems updates."
    );
    expect(tooltip).toBeInTheDocument();
  });
});
