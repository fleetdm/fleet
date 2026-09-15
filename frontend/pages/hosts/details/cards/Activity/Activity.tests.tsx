import { screen } from "@testing-library/react";
import React from "react";

import {
  createMockCommand,
  createMockGetCommandsResponse,
} from "__mocks__/commandMock";
import { createCustomRenderer } from "test/test-utils";

import Activity from "./Activity";

const ANDROID_TOOLTIP =
  "Activities and non-custom MDM commands are not supported yet for Android.";

const render = createCustomRenderer();

const defaultProps = {
  activeTab: "upcoming" as const,
  showMDMCommandsToggle: true,
  showMDMCommands: true,
  upcomingCount: 1,
  canCancelActivities: true,
  onChangeTab: jest.fn(),
  onNextPage: jest.fn(),
  onPreviousPage: jest.fn(),
  onShowDetails: jest.fn(),
  onShowCommandDetails: jest.fn(),
  onCancel: jest.fn(),
  onShowMDMCommands: jest.fn(),
  onHideMDMCommands: jest.fn(),
};

const pendingCommands = createMockGetCommandsResponse({
  count: 1,
  results: [
    createMockCommand({
      request_type: "REQUEST_DEVICE_INFO",
      command_status: "pending",
      status: "Pending",
    }),
  ],
});

describe("Activity card", () => {
  it("renders an enabled Upcoming tab with its count", () => {
    render(<Activity {...defaultProps} commands={pendingCommands} />);

    const upcomingTab = screen.getByRole("tab", { name: /upcoming/i });
    expect(upcomingTab).toBeInTheDocument();
    expect(upcomingTab).toHaveAttribute("aria-disabled", "false");
    expect(screen.getByText("1")).toBeInTheDocument();
  });

  it("lists pending MDM commands under the Upcoming tab", () => {
    render(<Activity {...defaultProps} commands={pendingCommands} />);

    expect(screen.getByText("REQUEST_DEVICE_INFO")).toBeInTheDocument();
    expect(screen.getByText(/is pending/)).toBeInTheDocument();
  });

  describe("MDM commands toggle", () => {
    it("is interactive by default", async () => {
      const onHideMDMCommands = jest.fn();
      const { user } = render(
        <Activity
          {...defaultProps}
          commands={pendingCommands}
          onHideMDMCommands={onHideMDMCommands}
        />
      );

      const [toggle] = screen.getAllByRole("switch");
      expect(toggle).toBeEnabled();
      expect(screen.queryByText(ANDROID_TOOLTIP)).not.toBeInTheDocument();

      await user.click(toggle);
      expect(onHideMDMCommands).toHaveBeenCalled();
    });

    it("can't be flipped and explains why when disabled", async () => {
      const onHideMDMCommands = jest.fn();
      const { user } = render(
        <Activity
          {...defaultProps}
          commands={pendingCommands}
          isMDMCommandsToggleDisabled
          mdmCommandsToggleTooltip={ANDROID_TOOLTIP}
          onHideMDMCommands={onHideMDMCommands}
        />
      );

      const [toggle] = screen.getAllByRole("switch");
      expect(toggle).toBeDisabled();
      expect(toggle).toBeChecked();

      await user.click(toggle);
      expect(onHideMDMCommands).not.toHaveBeenCalled();

      // the label stays hoverable while the switch is disabled so the
      // explanation is still reachable
      await user.hover(screen.getAllByText(/Show MDM commands/)[0]);
      expect(await screen.findByText(ANDROID_TOOLTIP)).toBeInTheDocument();
    });
  });
});
