import React from "react";
import { render, screen } from "@testing-library/react";
import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";

import DisabledHostOrbitDebugLoggingActivityItem from "./DisabledHostOrbitDebugLoggingActivityItem";

describe("DisabledHostOrbitDebugLoggingActivityItem", () => {
  it("renders the actor name and action text", () => {
    render(
      <DisabledHostOrbitDebugLoggingActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Test User",
          type: ActivityType.DisabledHostOrbitDebugLogging,
          details: {},
        })}
        tab="past"
      />
    );

    expect(screen.getByText("Test User")).toBeVisible();
    expect(
      screen.getByText(/disabled orbit debug logging on this host/i)
    ).toBeVisible();
  });

  it("does not render the cancel or show-details icons", () => {
    render(
      <DisabledHostOrbitDebugLoggingActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Test User",
          type: ActivityType.DisabledHostOrbitDebugLogging,
          details: {},
        })}
        tab="past"
      />
    );

    expect(screen.queryByTestId("close-icon")).not.toBeInTheDocument();
    expect(screen.queryByTestId("info-outline-icon")).not.toBeInTheDocument();
  });
});
