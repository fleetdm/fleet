import { render, screen } from "@testing-library/react";
import React from "react";

import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";

import FailedToRotateDiskEncryptionKeyActivityItem from "./FailedToRotateDiskEncryptionKey";

const failedActivity = (detail?: string) =>
  createMockHostPastActivity({
    actor_full_name: "Fleet",
    fleet_initiated: true,
    type: ActivityType.FailedToRotateDiskEncryptionKey,
    ...(detail === undefined ? {} : { details: { detail } }),
  });

describe("FailedToRotateDiskEncryptionKeyActivityItem", () => {
  it("renders the failure attributed to Fleet", () => {
    render(
      <FailedToRotateDiskEncryptionKeyActivityItem
        activity={failedActivity(
          "MDMErrorDomain (12): The password is incorrect."
        )}
        tab="past"
        onShowDetails={jest.fn()}
      />
    );

    expect(screen.getByText("Fleet")).toBeVisible();
    expect(
      screen.getByText(
        /failed to rotate the disk encryption key for this host/i
      )
    ).toBeVisible();
    expect(screen.queryByTestId("close-icon")).not.toBeInTheDocument();
    expect(screen.getByTestId("info-outline-icon")).toBeInTheDocument();
  });

  it("offers details even without a recorded reason", () => {
    render(
      <FailedToRotateDiskEncryptionKeyActivityItem
        activity={failedActivity()}
        tab="past"
        onShowDetails={jest.fn()}
      />
    );

    expect(screen.getByTestId("info-outline-icon")).toBeInTheDocument();
  });
});
