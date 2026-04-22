import React from "react";
import { render, screen } from "@testing-library/react";
import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";

import EnabledHostOrbitDebugLoggingActivityItem from "./EnabledHostOrbitDebugLoggingActivityItem";

describe("EnabledHostOrbitDebugLoggingActivityItem", () => {
  it("renders the actor name and action text", () => {
    render(
      <EnabledHostOrbitDebugLoggingActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Test User",
          type: ActivityType.EnabledHostOrbitDebugLogging,
          details: { expires_at: "2026-04-15T18:00:00Z" },
        })}
        tab="past"
      />
    );

    expect(screen.getByText("Test User")).toBeVisible();
    expect(
      screen.getByText(/enabled orbit debug logging on this host/i)
    ).toBeVisible();
  });

  it("renders the expiry timestamp when present", () => {
    render(
      <EnabledHostOrbitDebugLoggingActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Test User",
          type: ActivityType.EnabledHostOrbitDebugLogging,
          details: { expires_at: "2026-04-15T18:00:00Z" },
        })}
        tab="past"
      />
    );

    // internationalTimeFormat is locale-dependent, so just check that
    // something follows "until" — exact format is covered by the helper's
    // own tests.
    expect(screen.getByText(/until /i)).toBeVisible();
  });

  it("omits the expiry suffix when expires_at is missing", () => {
    render(
      <EnabledHostOrbitDebugLoggingActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Test User",
          type: ActivityType.EnabledHostOrbitDebugLogging,
          details: {},
        })}
        tab="past"
      />
    );

    expect(screen.queryByText(/until /i)).not.toBeInTheDocument();
  });

  it("does not render the cancel or show-details icons", () => {
    render(
      <EnabledHostOrbitDebugLoggingActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Test User",
          type: ActivityType.EnabledHostOrbitDebugLogging,
          details: {},
        })}
        tab="past"
      />
    );

    expect(screen.queryByTestId("close-icon")).not.toBeInTheDocument();
    expect(screen.queryByTestId("info-outline-icon")).not.toBeInTheDocument();
  });
});
