import React from "react";
import { render, screen } from "@testing-library/react";
import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";

import FailedToRotateManagedLocalAccountPasswordActivityItem from "./FailedToRotateManagedLocalAccountPassword";

describe("FailedToRotateManagedLocalAccountPasswordActivityItem", () => {
  it("renders Fleet-initiated failed rotation activity content", () => {
    render(
      <FailedToRotateManagedLocalAccountPasswordActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Fleet",
          fleet_initiated: true,
          type: ActivityType.FailedToRotateManagedLocalAccountPassword,
        })}
        tab="past"
      />
    );

    expect(screen.getByText("Fleet")).toBeVisible();
    expect(
      screen.getByText(
        /failed to rotate the managed local account password for this host/i
      )
    ).toBeVisible();
  });

  it("does not render the cancel icon", () => {
    render(
      <FailedToRotateManagedLocalAccountPasswordActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Fleet",
          fleet_initiated: true,
          type: ActivityType.FailedToRotateManagedLocalAccountPassword,
        })}
        tab="past"
      />
    );

    expect(screen.queryByTestId("close-icon")).not.toBeInTheDocument();
  });

  it("does not render the show details icon", () => {
    render(
      <FailedToRotateManagedLocalAccountPasswordActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Fleet",
          fleet_initiated: true,
          type: ActivityType.FailedToRotateManagedLocalAccountPassword,
        })}
        tab="past"
      />
    );

    expect(screen.queryByTestId("info-outline-icon")).not.toBeInTheDocument();
  });
});
