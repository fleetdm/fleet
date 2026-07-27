import React from "react";
import { render, screen } from "@testing-library/react";
import { createMockHostPastActivity } from "__mocks__/activityMock";

import { ActivityType, IHostPastActivityType } from "interfaces/activity";

import PolicyAutomationActivityItem from "./PolicyAutomationActivityItem";

const renderItem = (type: IHostPastActivityType) =>
  render(
    <PolicyAutomationActivityItem
      activity={createMockHostPastActivity({
        type,
        actor_full_name: "",
        actor_id: 0,
        fleet_initiated: true,
        details: {},
      })}
      tab="past"
    />
  );

describe("PolicyAutomationActivityItem", () => {
  const cases: Array<[IHostPastActivityType, RegExp]> = [
    [ActivityType.RanAutomationWebhook, /sent a webhook because this host/i],
    [ActivityType.RanAutomationTicket, /created a ticket because this host/i],
    [
      ActivityType.RanAutomationCalendarEvent,
      /created a calendar event because this host/i,
    ],
    [
      ActivityType.RanAutomationConditionalAccess,
      /blocked single sign-on because this host/i,
    ],
    [
      ActivityType.FailedAutomationWebhook,
      /failed to send a webhook after this host/i,
    ],
    [
      ActivityType.FailedAutomationTicket,
      /failed to create a ticket after this host/i,
    ],
    [
      ActivityType.FailedAutomationCalendarEvent,
      /failed to create a calendar event after this host/i,
    ],
    [
      ActivityType.FailedAutomationConditionalAccess,
      /failed to block single sign-on after this host/i,
    ],
  ];

  it.each(cases)("renders copy for %s", (type, expected) => {
    renderItem(type);
    expect(screen.getByText("Fleet")).toBeVisible();
    expect(screen.getByText(expected)).toBeVisible();
  });

  it("does not render the cancel or show-details icons", () => {
    renderItem(ActivityType.RanAutomationWebhook);
    expect(screen.queryByTestId("close-icon")).not.toBeInTheDocument();
    expect(screen.queryByTestId("info-outline-icon")).not.toBeInTheDocument();
  });

  // These activities are always Fleet-initiated in practice; this documents the
  // defensive actor branch that keeps the bold name in sync with the avatar.
  it("renders the actor name when the activity is not Fleet-initiated", () => {
    render(
      <PolicyAutomationActivityItem
        activity={createMockHostPastActivity({
          type: ActivityType.RanAutomationWebhook,
          actor_full_name: "Admin User",
          actor_id: 1,
          fleet_initiated: false,
          details: {},
        })}
        tab="past"
      />
    );
    expect(screen.getByText("Admin User")).toBeVisible();
    expect(screen.queryByText("Fleet")).not.toBeInTheDocument();
  });
});
