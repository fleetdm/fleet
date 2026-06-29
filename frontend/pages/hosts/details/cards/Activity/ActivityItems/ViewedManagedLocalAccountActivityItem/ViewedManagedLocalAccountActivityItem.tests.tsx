import React from "react";
import { render, screen } from "@testing-library/react";
import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";

import ViewedManagedLocalAccountActivityItem from "./ViewedManagedLocalAccountActivityItem";

describe("ViewedManagedLocalAccountActivityItem", () => {
  it("renders the activity content", () => {
    render(
      <ViewedManagedLocalAccountActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Test User",
          type: ActivityType.ViewedManagedLocalAccount,
        })}
        tab="past"
      />
    );

    expect(screen.getByText("Test User")).toBeVisible();
    expect(
      screen.getByText(/viewed the managed local account on this host/i)
    ).toBeVisible();
  });

  it("does not render the cancel icon", () => {
    render(
      <ViewedManagedLocalAccountActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Test User",
          type: ActivityType.ViewedManagedLocalAccount,
        })}
        tab="past"
      />
    );

    expect(screen.queryByTestId("close-icon")).not.toBeInTheDocument();
  });

  it("does not render the show details icon", () => {
    render(
      <ViewedManagedLocalAccountActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Test User",
          type: ActivityType.ViewedManagedLocalAccount,
        })}
        tab="past"
      />
    );

    expect(screen.queryByTestId("info-outline-icon")).not.toBeInTheDocument();
  });
});
