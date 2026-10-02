import { render, screen } from "@testing-library/react";
import React from "react";

import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";

import RotatedDiskEncryptionKeyActivityItem from "./RotatedDiskEncryptionKey";

describe("RotatedDiskEncryptionKeyActivityItem", () => {
  it("renders the user who triggered the rotation", () => {
    render(
      <RotatedDiskEncryptionKeyActivityItem
        activity={createMockHostPastActivity({
          actor_full_name: "Test User",
          type: ActivityType.RotatedDiskEncryptionKey,
        })}
        tab="past"
      />
    );

    expect(screen.getByText("Test User")).toBeVisible();
    expect(
      screen.getByText(
        /triggered rotation of the disk encryption key on this host/i
      )
    ).toBeVisible();
    expect(screen.queryByTestId("close-icon")).not.toBeInTheDocument();
    expect(screen.queryByTestId("info-outline-icon")).not.toBeInTheDocument();
  });
});
