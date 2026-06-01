import React from "react";
import { render, screen } from "@testing-library/react";
import { createMockHostPastActivity } from "__mocks__/activityMock";

import { ActivityType } from "interfaces/activity";

import MdmUnenrolledActivityItem from "./MdmUnenrolledActivityItem";

describe("MdmUnenrolledActivityItem", () => {
  it("renders admin-initiated unenrollment copy for Apple/macOS", () => {
    render(
      <MdmUnenrolledActivityItem
        activity={createMockHostPastActivity({
          type: ActivityType.MdmUnenrolled,
          actor_full_name: "Admin User",
          details: { platform: "darwin" },
        })}
        tab="past"
      />
    );

    expect(screen.getByText("Admin User")).toBeVisible();
    expect(
      screen.getByText(
        /told Fleet to turn off mobile device management \(MDM\) for this host/i
      )
    ).toBeVisible();
  });

  it("renders end-user-initiated unenrollment copy for macOS (no actor)", () => {
    render(
      <MdmUnenrolledActivityItem
        activity={createMockHostPastActivity({
          type: ActivityType.MdmUnenrolled,
          actor_full_name: "",
          actor_id: 0,
          details: { platform: "darwin" },
        })}
        tab="past"
      />
    );

    expect(
      screen.getByText(
        /Mobile device management \(MDM\) was turned off for this host/i
      )
    ).toBeVisible();
  });

  it("renders Android-specific copy when actor is set", () => {
    render(
      <MdmUnenrolledActivityItem
        activity={createMockHostPastActivity({
          type: ActivityType.MdmUnenrolled,
          actor_full_name: "Admin User",
          details: { platform: "android" },
        })}
        tab="past"
      />
    );

    expect(screen.getByText("Admin User")).toBeVisible();
    expect(screen.getByText(/told Fleet to unenroll this host/i)).toBeVisible();
  });

  it("renders Android-specific copy when no actor", () => {
    render(
      <MdmUnenrolledActivityItem
        activity={createMockHostPastActivity({
          type: ActivityType.MdmUnenrolled,
          actor_full_name: "",
          actor_id: 0,
          details: { platform: "android" },
        })}
        tab="past"
      />
    );

    expect(
      screen.getByText(/This host is unenrolled from Fleet/i)
    ).toBeVisible();
  });

  it("does not render the cancel or show-details icons", () => {
    render(
      <MdmUnenrolledActivityItem
        activity={createMockHostPastActivity({
          type: ActivityType.MdmUnenrolled,
          actor_full_name: "Admin User",
          details: { platform: "darwin" },
        })}
        tab="past"
      />
    );

    expect(screen.queryByTestId("close-icon")).not.toBeInTheDocument();
    expect(screen.queryByTestId("info-outline-icon")).not.toBeInTheDocument();
  });
});
