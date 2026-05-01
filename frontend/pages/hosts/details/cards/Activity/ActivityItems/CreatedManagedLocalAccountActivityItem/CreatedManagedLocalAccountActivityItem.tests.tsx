import React from "react";
import { render, screen } from "@testing-library/react";
import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";

import CreatedManagedLocalAccountActivityItem from "./CreatedManagedLocalAccountActivityItem";

describe("CreatedManagedLocalAccountActivityItem", () => {
  it("renders the activity content", () => {
    render(
      <CreatedManagedLocalAccountActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Fleet",
          type: ActivityType.CreatedManagedLocalAccount,
        })}
        tab="past"
      />
    );

    expect(screen.getByText("Fleet")).toBeVisible();
    expect(
      screen.getByText(/created a managed local account for this host/i)
    ).toBeVisible();
  });

  it("does not render the cancel icon", () => {
    render(
      <CreatedManagedLocalAccountActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Fleet",
          type: ActivityType.CreatedManagedLocalAccount,
        })}
        tab="past"
      />
    );

    expect(screen.queryByTestId("close-icon")).not.toBeInTheDocument();
  });

  it("does not render the show details icon", () => {
    render(
      <CreatedManagedLocalAccountActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Fleet",
          type: ActivityType.CreatedManagedLocalAccount,
        })}
        tab="past"
      />
    );

    expect(screen.queryByTestId("info-outline-icon")).not.toBeInTheDocument();
  });
});
