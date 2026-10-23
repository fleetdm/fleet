import { render, screen } from "@testing-library/react";
import React from "react";

import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";

import DeletedHostGoogleDeviceIdActivityItem from "./DeletedHostGoogleDeviceIdActivityItem";

describe("DeletedHostGoogleDeviceIdActivityItem", () => {
  const activity = createMockHostPastActivity({
    type: ActivityType.DeletedHostGoogleDeviceId,
    actor_full_name: "Test User",
    details: {
      host_display_name: "Anna's iPhone",
      google_device_id: "c3d1a9e2-7b4f-4e8a-9f1d-2a6b8c0e4f12",
    },
  });

  it("renders the actor and the Google device ID", () => {
    render(
      <DeletedHostGoogleDeviceIdActivityItem activity={activity} tab="past" />
    );

    expect(screen.getByText("Test User")).toBeVisible();
    expect(screen.getByText(/deleted the Google device ID/i)).toBeVisible();
    expect(
      screen.getByText("c3d1a9e2-7b4f-4e8a-9f1d-2a6b8c0e4f12")
    ).toBeVisible();
  });

  it("does not render the cancel or show details icons", () => {
    render(
      <DeletedHostGoogleDeviceIdActivityItem activity={activity} tab="past" />
    );

    expect(screen.queryByTestId("close-icon")).not.toBeInTheDocument();
    expect(screen.queryByTestId("info-outline-icon")).not.toBeInTheDocument();
  });
});
