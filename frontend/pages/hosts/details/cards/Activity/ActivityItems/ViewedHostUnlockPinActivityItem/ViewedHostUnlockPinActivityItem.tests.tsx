import { render, screen } from "@testing-library/react";
import React from "react";

import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";

import ViewedHostUnlockPinActivityItem from "./ViewedHostUnlockPinActivityItem";

describe("ViewedHostUnlockPinActivityItem", () => {
  const renderItem = (details?: Record<string, string>) =>
    render(
      <ViewedHostUnlockPinActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Test User",
          type: ActivityType.ViewedHostUnlockPin,
          details,
        })}
        tab="past"
      />
    );

  it("renders the actor and host display name", () => {
    renderItem({ host_display_name: "Test Samsung" });

    expect(screen.getByText("Test User")).toBeVisible();
    expect(screen.getByText("Test Samsung")).toBeVisible();
    expect(screen.getByText(/viewed the unlock PIN for/i)).toBeVisible();
  });

  it("falls back to 'this host' without a host display name", () => {
    renderItem();

    expect(
      screen.getByText(/viewed the unlock PIN for this host\./i)
    ).toBeVisible();
  });

  it("does not render the cancel or show details icons", () => {
    renderItem({ host_display_name: "Test Samsung" });

    expect(screen.queryByTestId("close-icon")).not.toBeInTheDocument();
    expect(screen.queryByTestId("info-icon")).not.toBeInTheDocument();
  });
});
