import React from "react";
import { screen } from "@testing-library/react";
import { noop } from "lodash";

import { createCustomRenderer } from "test/test-utils";

import PlatformTabs from "./PlatformTabs";

const render = createCustomRenderer({ withBackendMock: true });

const defaultProps = {
  currentTeamId: 1,
  defaultMacOSVersion: "11.0",
  defaultMacOSDeadline: "2024-12-31",
  defaultMacOSDeadlineDays: "",
  defaultMacOSUpdateNewHosts: true,
  defaultIOSVersion: "17.5",
  defaultIOSDeadline: "2024-12-31",
  defaultIOSDeadlineDays: "",
  defaultIPadOSVersion: "18.5",
  defaultIPadOSDeadline: "2024-12-31",
  defaultIPadOSDeadlineDays: "",
  defaultWindowsDeadlineDays: "5",
  defaultWindowsGracePeriodDays: "2",
  onSelectPlatform: noop,
  refetchAppConfig: noop,
  refetchTeamConfig: noop,
  isWindowsMdmEnabled: true,
  isAndroidMdmEnabled: true,
};

describe("PlatformTabs", () => {
  // Only the Apple forms offer a target to choose; Windows is always deadline
  // driven and Android isn't supported yet. The tabs decide which form each
  // platform gets, so the dropdown must not leak into the other two.
  it("renders the target dropdown on the macOS tab", () => {
    render(<PlatformTabs {...defaultProps} selectedPlatform="darwin" />);

    expect(screen.getByLabelText(/Target/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Minimum version/i)).toBeInTheDocument();
  });

  it("renders the target dropdown on the iOS tab", () => {
    render(<PlatformTabs {...defaultProps} selectedPlatform="ios" />);

    expect(screen.getByLabelText(/Target/i)).toBeInTheDocument();
  });

  it("renders the target dropdown on the iPadOS tab", () => {
    render(<PlatformTabs {...defaultProps} selectedPlatform="ipados" />);

    expect(screen.getByLabelText(/Target/i)).toBeInTheDocument();
  });

  it("does not render the target dropdown on the Windows tab", () => {
    render(<PlatformTabs {...defaultProps} selectedPlatform="windows" />);

    expect(screen.queryByLabelText(/Target/i)).not.toBeInTheDocument();
    // Windows fields don't associate their labels with the input, so match the
    // label text rather than the control.
    expect(screen.getByText(/Grace period/i)).toBeInTheDocument();
  });

  it("does not render the target dropdown on the Android tab", () => {
    render(<PlatformTabs {...defaultProps} selectedPlatform="android" />);

    expect(screen.queryByLabelText(/Target/i)).not.toBeInTheDocument();
    expect(screen.getByText(/Android updates are coming soon/i)).toBeVisible();
  });

  it("hides the Windows and Android tabs when their MDM isn't enabled", () => {
    render(
      <PlatformTabs
        {...defaultProps}
        selectedPlatform="darwin"
        isWindowsMdmEnabled={false}
        isAndroidMdmEnabled={false}
      />
    );

    expect(screen.getByRole("tab", { name: /macOS/i })).toBeInTheDocument();
    expect(
      screen.queryByRole("tab", { name: /Windows/i })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("tab", { name: /Android/i })
    ).not.toBeInTheDocument();
  });
});
