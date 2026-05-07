import React from "react";
import { render, screen } from "@testing-library/react";
import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";

import RotatedManagedLocalAccountPasswordActivityItem from "./RotatedManagedLocalAccountPassword";

describe("RotatedManagedLocalAccountPasswordActivityItem", () => {
  it("renders user-triggered rotation activity content", () => {
    render(
      <RotatedManagedLocalAccountPasswordActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Test User",
          type: ActivityType.RotatedManagedLocalAccountPassword,
        })}
        tab="past"
      />
    );

    expect(screen.getByText("Test User")).toBeVisible();
    expect(
      screen.getByText(
        /triggered rotation of the managed local account password/i
      )
    ).toBeVisible();
  });

  it("renders Fleet-initiated (auto) rotation activity content", () => {
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
        /triggered rotation of the managed local account password/i
      )
    ).toBeVisible();
  });

  it("does not render the cancel icon", () => {
    render(
      <RotatedManagedLocalAccountPasswordActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Test User",
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
          actor_full_name: "Test User",
          type: ActivityType.RotatedManagedLocalAccountPassword,
        })}
        tab="past"
      />
    );

    expect(screen.queryByTestId("info-outline-icon")).not.toBeInTheDocument();
  });
});
