import React from "react";
import { render, screen } from "@testing-library/react";
import { createMockHostPastActivity } from "__mocks__/activityMock";

import UnlockHostActivityItem from "./UnlockedHostActivityItem";

describe("UnlockHostActivityItem", () => {
  it("renders the activity content for darwin hosts", () => {
    render(
      <UnlockHostActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Test User",
          details: { host_platform: "darwin" },
        })}
        tab="past"
      />
    );

    expect(screen.getByText("Test User")).toBeVisible();
    expect(
      screen.getByText("viewed the six-digit unlock PIN for this host.")
    ).toBeVisible();
  });

  it("renders the activity content for non-darwin hosts", () => {
    render(
      <UnlockHostActivityItem
        activity={createMockHostPastActivity({ actor_full_name: "Test User" })}
        tab="past"
      />
    );

    expect(screen.getByText("Test User")).toBeVisible();
    expect(screen.getByText(/unlocked this host/i)).toBeVisible();
  });

  it("does not render the cancel icon", () => {
    render(
      <UnlockHostActivityItem
        activity={createMockHostPastActivity({ actor_full_name: "Test User" })}
        tab="past"
      />
    );

    expect(screen.queryByTestId("close-icon")).not.toBeInTheDocument();
  });

  it("does not render the show details icon", () => {
    render(
      <UnlockHostActivityItem
        activity={createMockHostPastActivity({ actor_full_name: "Test User" })}
        tab="past"
      />
    );

    expect(screen.queryByTestId("info-outline-icon")).not.toBeInTheDocument();
  });
});
