import React from "react";
import { render, screen } from "@testing-library/react";
import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";

import ResetPolicyActivityItem from "./ResetPolicyActivityItem";

describe("ResetPolicyActivityItem", () => {
  const activity = createMockHostPastActivity({
    type: ActivityType.ResetPolicy,
    actor_full_name: "Test User",
    details: { policy_name: "Test policy", host_id: 1 },
  });

  it("renders the actor and policy name", () => {
    render(<ResetPolicyActivityItem activity={activity} tab="past" />);

    expect(screen.getByText("Test User")).toBeVisible();
    expect(screen.getByText("Test policy")).toBeVisible();
    expect(screen.getByText(/reset the policy/i)).toBeVisible();
    expect(screen.getByText(/for this host/i)).toBeVisible();
  });

  it("does not render the cancel or show details icons", () => {
    render(<ResetPolicyActivityItem activity={activity} tab="past" />);

    expect(screen.queryByTestId("close-icon")).not.toBeInTheDocument();
    expect(screen.queryByTestId("info-outline-icon")).not.toBeInTheDocument();
  });
});
