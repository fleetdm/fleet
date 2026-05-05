import React from "react";
import { render, screen } from "@testing-library/react";

import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";

import CanceledUninstallSoftwareActivtyItem from "./CanceledUninstallSoftwareActivtyItem";

describe("CanceledUninstallSoftwareActivityItem", () => {
  const mockActivity = createMockHostPastActivity({
    type: ActivityType.CanceledUninstallSoftware,
    details: { software_title: "test.sh" },
  });

  it("renders the activity content", () => {
    render(
      <CanceledUninstallSoftwareActivtyItem
        tab="past"
        activity={mockActivity}
      />
    );

    expect(screen.getByText("Test User")).toBeVisible();
    expect(screen.getByText(/canceled/i)).toBeVisible();
    expect(screen.getByText(/test.sh/i)).toBeVisible();
    expect(screen.getByText(/uninstall on this host/i)).toBeVisible();
  });

  it("does not render the cancel icon", () => {
    render(
      <CanceledUninstallSoftwareActivtyItem
        tab="past"
        activity={mockActivity}
      />
    );

    expect(screen.queryByTestId("close-icon")).not.toBeInTheDocument();
  });

  it("does not render the show details icon", () => {
    render(
      <CanceledUninstallSoftwareActivtyItem
        tab="past"
        activity={mockActivity}
      />
    );

    expect(screen.queryByTestId("info-outline-icon")).not.toBeInTheDocument();
  });
});
