import React from "react";
import { render, screen } from "@testing-library/react";
import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";

import EditedCustomHostVitalValueActivityItem from "./EditedCustomHostVitalValueActivityItem";

describe("EditedCustomHostVitalValueActivityItem", () => {
  const activity = createMockHostPastActivity({
    actor_full_name: "Test User",
    type: ActivityType.EditedCustomHostVitalValue,
    details: { custom_host_vital_name: "Asset tag" },
  });

  it("renders the activity content", () => {
    render(
      <EditedCustomHostVitalValueActivityItem activity={activity} tab="past" />
    );

    expect(screen.getByText("Test User")).toBeVisible();
    expect(screen.getByText("Asset tag")).toBeVisible();
  });

  it("does not render the cancel icon", () => {
    render(
      <EditedCustomHostVitalValueActivityItem activity={activity} tab="past" />
    );

    expect(screen.queryByTestId("close-icon")).not.toBeInTheDocument();
  });

  it("does not render the show details icon", () => {
    render(
      <EditedCustomHostVitalValueActivityItem activity={activity} tab="past" />
    );

    expect(screen.queryByTestId("info-outline-icon")).not.toBeInTheDocument();
  });
});
