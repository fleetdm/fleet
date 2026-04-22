import React from "react";
import { render, screen } from "@testing-library/react";
import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";

import RotatedManagedLocalAccountPasswordActivityItem from "./RotatedManagedLocalAccountPasswordActivityItem";

describe("RotatedManagedLocalAccountPasswordActivityItem", () => {
  it("renders the activity content", () => {
    render(
      <RotatedManagedLocalAccountPasswordActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Fleet",
          type: ActivityType.RotatedManagedLocalAccountPassword,
        })}
        tab="past"
      />
    );

    expect(screen.getByText("Fleet")).toBeVisible();
    expect(
      screen.getByText(
        /rotated the managed local account password for this host/i
      )
    ).toBeVisible();
  });

  it("does not render the cancel icon", () => {
    render(
      <RotatedManagedLocalAccountPasswordActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Fleet",
          type: ActivityType.RotatedManagedLocalAccountPassword,
        })}
        tab="past"
      />
    );

    expect(screen.queryByTestId("close-icon")).not.toBeInTheDocument();
  });

  it("does not render the show details icon", () => {
    render(
      <RotatedManagedLocalAccountPasswordActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Fleet",
          type: ActivityType.RotatedManagedLocalAccountPassword,
        })}
        tab="past"
      />
    );

    expect(screen.queryByTestId("info-outline-icon")).not.toBeInTheDocument();
  });
});
