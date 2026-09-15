import { screen, within } from "@testing-library/react";
import React from "react";

import {
  createMockCommand,
  createMockGetCommandsResponse,
} from "__mocks__/commandMock";
import { createCustomRenderer } from "test/test-utils";

import Activity from "./Activity";

const TOGGLE_TOOLTIP = "Not supported on this platform.";

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
      request_type: "LOCK",
      command_status: "pending",
      status: "Pending",
    }),
  ],
});

describe("Activity card", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("renders an enabled Upcoming tab with its count", () => {
    render(<Activity {...defaultProps} commands={pendingCommands} />);

    const upcomingTab = screen.getByRole("tab", { name: /upcoming/i });
    expect(upcomingTab).toHaveAttribute("aria-disabled", "false");
    expect(within(upcomingTab).getByText("1")).toBeInTheDocument();
  });

  it("lists pending MDM commands under the Upcoming tab", () => {
    render(<Activity {...defaultProps} commands={pendingCommands} />);

    expect(screen.getByText("LOCK")).toBeInTheDocument();
    expect(screen.getByText(/is pending/)).toBeInTheDocument();
  });

  it("does not offer to cancel a command that isn't cancelable", () => {
    render(
      <Activity
        {...defaultProps}
        commands={pendingCommands}
        onCancelCommand={jest.fn()}
      />
    );

    // Android request types aren't in CANCELABLE_REQUEST_TYPES, and the server
    // only cancels Apple commands
    expect(
      screen.queryByRole("button", { name: /cancel/i })
    ).not.toBeInTheDocument();
  });

  it("never renders a command feed without the toggle that governs it", () => {
    // the commands queries keep previous data across host navigations, so
    // `commands` can hold a stale response for a host with no toggle
    render(
      <Activity
        {...defaultProps}
        showMDMCommandsToggle={false}
        commands={pendingCommands}
        activities={{
          activities: [],
          count: 0,
          meta: { has_next_results: false, has_previous_results: false },
        }}
      />
    );

    expect(screen.queryAllByRole("switch")).toHaveLength(0);
    expect(screen.queryByText("LOCK")).not.toBeInTheDocument();
    expect(screen.getByText(/No pending activity/)).toBeInTheDocument();
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
      expect(screen.queryByText(TOGGLE_TOOLTIP)).not.toBeInTheDocument();

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
          mdmCommandsToggleTooltip={TOGGLE_TOOLTIP}
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
      expect(await screen.findByText(TOGGLE_TOOLTIP)).toBeInTheDocument();
    });
  });
});
