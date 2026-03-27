import React from "react";
import { render, screen } from "@testing-library/react";
import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";

import RotatedHostRecoveryLockPasswordActivityItem from "./RotatedHostRecoveryLockPassword";

describe("RotatedHostRecoveryLockPasswordActivityItem", () => {
  it("renders user-triggered rotation activity content", () => {
    render(
      <RotatedHostRecoveryLockPasswordActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Test User",
          type: ActivityType.RotatedHostRecoveryLockPassword,
        })}
        tab="past"
      />
    );

    expect(screen.getByText("Test User")).toBeVisible();
    expect(
      screen.getByText(/triggered rotation of the Recovery Lock password/i)
    ).toBeVisible();
  });

  it("renders Fleet-initiated (auto) rotation activity content", () => {
    render(
      <RotatedHostRecoveryLockPasswordActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Fleet",
          type: ActivityType.RotatedHostRecoveryLockPassword,
        })}
        tab="past"
      />
    );

    expect(screen.getByText("Fleet")).toBeVisible();
    expect(
      screen.getByText(/triggered rotation of the Recovery Lock password/i)
    ).toBeVisible();
  });

  it("does not render the cancel icon", () => {
    render(
      <RotatedHostRecoveryLockPasswordActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Test User",
          type: ActivityType.RotatedHostRecoveryLockPassword,
        })}
        tab="past"
      />
    );

    expect(screen.queryByTestId("close-icon")).not.toBeInTheDocument();
  });

  it("does not render the show details icon", () => {
    render(
      <RotatedHostRecoveryLockPasswordActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Test User",
          type: ActivityType.RotatedHostRecoveryLockPassword,
        })}
        tab="past"
      />
    );

    expect(screen.queryByTestId("info-outline-icon")).not.toBeInTheDocument();
  });
});
