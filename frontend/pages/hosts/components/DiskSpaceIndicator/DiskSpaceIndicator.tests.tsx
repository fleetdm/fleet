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

  it("renders 'Not supported' text when disk space is sentinel value -1", () => {
    const { container } = renderWithSetup(
      <DiskSpaceIndicator
        gigsDiskSpaceAvailable={-1}
        percentDiskSpaceAvailable={0}
        platform="android"
        tooltipPosition="bottom"
      />
    );

    const notSupportedElement = container.querySelector(".not-supported");
    expect(notSupportedElement).toBeInTheDocument();
    expect(notSupportedElement).toHaveTextContent("Not supported");
  });

  it("distinguishes between zero storage (disk full) and unsupported storage", () => {
    // Case 1: Zero storage should show "No data available"
    const { container: zeroContainer, rerender } = renderWithSetup(
      <DiskSpaceIndicator
        gigsDiskSpaceAvailable={0}
        percentDiskSpaceAvailable={0}
        platform="android"
        tooltipPosition="bottom"
      />
    );

    const emptyElement = zeroContainer.querySelector(
      ".disk-space-indicator__empty"
    );
    expect(emptyElement).toBeInTheDocument();
    expect(emptyElement).toHaveTextContent("No data available");

    const notSupportedElement = zeroContainer.querySelector(".not-supported");
    expect(notSupportedElement).not.toBeInTheDocument();

    // Case 2: Sentinel value -1 should show "Not supported"
    rerender(
      <DiskSpaceIndicator
        gigsDiskSpaceAvailable={-1}
        percentDiskSpaceAvailable={0}
        platform="android"
        tooltipPosition="bottom"
      />
    );

    expect(screen.getByText("Not supported")).toBeInTheDocument();
    expect(screen.queryByText("No data available")).not.toBeInTheDocument();
  });

  it("handles negative values other than -1 as unsupported", () => {
    const negativeValues = [-2, -10, -100];

    negativeValues.forEach((value) => {
      const { container } = renderWithSetup(
        <DiskSpaceIndicator
          gigsDiskSpaceAvailable={value}
          percentDiskSpaceAvailable={0}
          platform="android"
          tooltipPosition="bottom"
        />
      );

      const notSupportedElement = container.querySelector(".not-supported");
      expect(notSupportedElement).toBeInTheDocument();
      expect(notSupportedElement).toHaveTextContent("Not supported");
    });
  });
  it("renders warning tooltip for <32gB when hovering over the yellow disk space indicator for darwin or windows", async () => {
    const { user } = renderWithSetup(
      <DiskSpaceIndicator
        gigsDiskSpaceAvailable={17}
        percentDiskSpaceAvailable={10}
        platform="darwin"
        tooltipPosition="bottom"
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
        tooltipPosition="bottom"
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
        tooltipPosition="bottom"
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
  it("renders tooltip over anchor for Linux hosts with gigs all disk space and gigs total disk space", async () => {
    const { user } = renderWithSetup(
      <DiskSpaceIndicator
        gigsDiskSpaceAvailable={20}
        percentDiskSpaceAvailable={20}
        gigsTotalDiskSpace={100}
        gigsAllDiskSpace={120}
        platform="linux"
      />
    );

    await user.hover(screen.getByText(/GB/));

    await waitFor(() => {
      const totalTip = screen.getByText(/System disk space/);
      expect(totalTip).toBeInTheDocument();
    });

    const allTip = screen.getByText(/All partitions/);
    expect(allTip).toBeInTheDocument();
  });
});
