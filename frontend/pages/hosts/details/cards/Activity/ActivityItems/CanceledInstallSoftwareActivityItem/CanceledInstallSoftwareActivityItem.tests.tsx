import React from "react";
import { render, screen } from "@testing-library/react";

import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";

import CanceledInstallSoftwareActivityItem from "./CanceledInstallSoftwareActivityItem";

describe("CanceledInstallSoftwareActivityItem", () => {
  const mockActivity = createMockHostPastActivity({
    type: ActivityType.CanceledInstallSoftware,
    details: { software_title: "test.app" },
  });

  it("renders the activity content", () => {
    render(
      <CanceledInstallSoftwareActivityItem tab="past" activity={mockActivity} />
    );

    expect(screen.getByText("Test User")).toBeVisible();
    expect(screen.getByText(/canceled/i)).toBeVisible();
    expect(screen.getByText(/test.app/i)).toBeVisible();
    expect(screen.getByText(/install on this host/i)).toBeVisible();
  });

  it("does not render the cancel icon", () => {
    render(
      <CanceledInstallSoftwareActivityItem tab="past" activity={mockActivity} />
    );

    expect(screen.queryByTestId("close-icon")).not.toBeInTheDocument();
  });

  it("does not render the show details icon", () => {
    render(
      <CanceledInstallSoftwareActivityItem tab="past" activity={mockActivity} />
    );

    expect(screen.queryByTestId("info-outline-icon")).not.toBeInTheDocument();
  });
});
