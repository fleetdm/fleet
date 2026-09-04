import React from "react";
import { render, screen } from "@testing-library/react";

import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";

import CanceledMdmCommandActivityItem from "./CanceledMdmCommandActivityItem";

describe("CanceledMdmCommandActivityItem", () => {
  const mockActivity = createMockHostPastActivity({
    type: ActivityType.CanceledMdmCommand,
    details: { command_type: "DeviceLock" },
  });

  it("renders the activity content", () => {
    render(
      <CanceledMdmCommandActivityItem tab="past" activity={mockActivity} />
    );

    expect(screen.getByText("Test User")).toBeVisible();
    expect(screen.getByText(/canceled the pending/i)).toBeVisible();
    expect(screen.getByText("DeviceLock")).toBeVisible();
    expect(screen.getByText(/command on this host/i)).toBeVisible();
  });

  it("does not render the cancel icon", () => {
    render(
      <CanceledMdmCommandActivityItem tab="past" activity={mockActivity} />
    );

    expect(screen.queryByTestId("close-icon")).not.toBeInTheDocument();
  });

  it("does not render the show details icon", () => {
    render(
      <CanceledMdmCommandActivityItem tab="past" activity={mockActivity} />
    );

    expect(screen.queryByTestId("info-outline-icon")).not.toBeInTheDocument();
  });
});
