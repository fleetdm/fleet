import React from "react";
import { render, screen } from "@testing-library/react";
import { createMockHostPastActivity } from "__mocks__/activityMock";

import RotatedHostRecoveryLockPasswordActivityItem from "./RotatedHostRecoveryLockPassword";

describe("RotatedHostRecoveryLockPasswordActivityItem", () => {
  it("renders the activity content", () => {
    render(
      <RotatedHostRecoveryLockPasswordActivityItem
        activity={createMockHostPastActivity({ actor_full_name: "Test User" })}
        tab="past"
      />
    );

    expect(screen.getByText("Test User")).toBeVisible();
    expect(
      screen.getByText(/rotated the Recovery Lock password/i)
    ).toBeVisible();
  });

  it("does not render the cancel icon", () => {
    render(
      <RotatedHostRecoveryLockPasswordActivityItem
        activity={createMockHostPastActivity({ actor_full_name: "Test User" })}
        tab="past"
      />
    );

    expect(screen.queryByTestId("close-icon")).not.toBeInTheDocument();
  });

  it("does not render the show details icon", () => {
    render(
      <RotatedHostRecoveryLockPasswordActivityItem
        activity={createMockHostPastActivity({ actor_full_name: "Test User" })}
        tab="past"
      />
    );

    expect(screen.queryByTestId("info-outline-icon")).not.toBeInTheDocument();
  });
});
