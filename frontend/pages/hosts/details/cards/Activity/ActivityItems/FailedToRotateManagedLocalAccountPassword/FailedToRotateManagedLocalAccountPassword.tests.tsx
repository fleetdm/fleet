import React from "react";
import { render, screen } from "@testing-library/react";
import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";

import FailedToRotateManagedLocalAccountPasswordActivityItem from "./FailedToRotateManagedLocalAccountPassword";

const failedActivity = (detail?: string) =>
  createMockHostPastActivity({
    actor_full_name: "Fleet",
    fleet_initiated: true,
    type: ActivityType.FailedToRotateManagedLocalAccountPassword,
    ...(detail === undefined ? {} : { details: { detail } }),
  });

describe("FailedToRotateManagedLocalAccountPasswordActivityItem", () => {
  it("renders Fleet-initiated failed rotation activity content", () => {
    render(
      <FailedToRotateManagedLocalAccountPasswordActivityItem
        activity={failedActivity()}
        tab="past"
        onShowDetails={jest.fn()}
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
        activity={failedActivity()}
        tab="past"
        onShowDetails={jest.fn()}
      />
    );

    expect(screen.queryByTestId("close-icon")).not.toBeInTheDocument();
  });

  // The reason, not the platform, is what gates the icon. macOS never reports one, so it never gets an icon, and a
  // Windows failure that arrives without one must not offer an empty modal.
  it("does not render the show details icon when the host reported no reason", () => {
    render(
      <FailedToRotateManagedLocalAccountPasswordActivityItem
        activity={failedActivity()}
        tab="past"
        onShowDetails={jest.fn()}
      />
    );

    expect(screen.queryByTestId("info-outline-icon")).not.toBeInTheDocument();
  });

  it("renders the show details icon when the host reported a reason", () => {
    render(
      <FailedToRotateManagedLocalAccountPasswordActivityItem
        activity={failedActivity("NERR_PasswordTooShort")}
        tab="past"
        onShowDetails={jest.fn()}
      />
    );

    expect(screen.getByTestId("info-outline-icon")).toBeInTheDocument();
  });
});
