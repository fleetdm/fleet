import React from "react";
import { render, screen } from "@testing-library/react";
import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";

import ReadManagedLocalAccountActivityItem from "./ReadManagedLocalAccountActivityItem";

describe("ReadManagedLocalAccountActivityItem", () => {
  it("renders the activity content", () => {
    render(
      <ReadManagedLocalAccountActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Test User",
          type: ActivityType.ReadManagedLocalAccount,
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
      <ReadManagedLocalAccountActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Test User",
          type: ActivityType.ReadManagedLocalAccount,
        })}
        tab="past"
      />
    );

    expect(screen.queryByTestId("close-icon")).not.toBeInTheDocument();
  });

  it("does not render the show details icon", () => {
    render(
      <ReadManagedLocalAccountActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Test User",
          type: ActivityType.ReadManagedLocalAccount,
        })}
        tab="past"
      />
    );

    expect(screen.queryByTestId("info-outline-icon")).not.toBeInTheDocument();
  });
});
