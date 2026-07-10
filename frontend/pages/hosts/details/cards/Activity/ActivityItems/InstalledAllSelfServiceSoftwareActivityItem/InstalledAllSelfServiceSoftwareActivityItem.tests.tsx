import React from "react";
import { render, screen } from "@testing-library/react";
import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";

import InstalledAllSelfServiceSoftwareActivityItem from "./InstalledAllSelfServiceSoftwareActivityItem";

describe("InstalledAllSelfServiceSoftwareActivityItem", () => {
  it("renders the un-scoped roll-up as an end-user action", () => {
    render(
      <InstalledAllSelfServiceSoftwareActivityItem
        activity={createMockHostPastActivity({
          type: ActivityType.InstalledAllSelfServiceSoftware,
          actor_full_name: "Test User",
          details: {},
        })}
        tab="past"
      />
    );

    expect(screen.getByText("End user")).toBeVisible();
    expect(
      screen.getByText(/installed all the software in self-service/i)
    ).toBeVisible();
    // The actor is dropped in favor of "End user".
    expect(screen.queryByText("Test User")).not.toBeInTheDocument();
  });

  it("treats a null category name the same as un-scoped", () => {
    render(
      <InstalledAllSelfServiceSoftwareActivityItem
        activity={createMockHostPastActivity({
          type: ActivityType.InstalledAllSelfServiceSoftware,
          details: { self_service_category_name: null },
        })}
        tab="past"
      />
    );

    expect(
      screen.getByText(/installed all the software in self-service/i)
    ).toBeVisible();
  });

  it("names the category when the roll-up is category-scoped", () => {
    render(
      <InstalledAllSelfServiceSoftwareActivityItem
        activity={createMockHostPastActivity({
          type: ActivityType.InstalledAllSelfServiceSoftware,
          details: { self_service_category_name: "Productivity" },
        })}
        tab="past"
      />
    );

    expect(screen.getByText("End user")).toBeVisible();
    expect(screen.getByText("Install all")).toBeVisible();
    expect(screen.getByText("Productivity")).toBeVisible();
    expect(screen.getByText(/in the self-service/i)).toBeVisible();
  });

  it("does not render the cancel or show details icons", () => {
    render(
      <InstalledAllSelfServiceSoftwareActivityItem
        activity={createMockHostPastActivity({
          type: ActivityType.InstalledAllSelfServiceSoftware,
          details: {},
        })}
        tab="past"
      />
    );

    expect(screen.queryByTestId("close-icon")).not.toBeInTheDocument();
    expect(screen.queryByTestId("info-outline-icon")).not.toBeInTheDocument();
  });
});
