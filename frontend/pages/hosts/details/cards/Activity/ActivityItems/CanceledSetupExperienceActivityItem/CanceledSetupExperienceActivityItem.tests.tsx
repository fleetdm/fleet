import React from "react";
import { render, screen } from "@testing-library/react";

import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";

import CanceledSetupExperienceActivityItem from "./CanceledSetupExperienceActivityItem";

describe("CanceledSetupExperienceActivityItem", () => {
  it("renders the software's display name instead of its raw name when set", () => {
    const activity = createMockHostPastActivity({
      type: ActivityType.CanceledSetupExperience,
      actor_full_name: "Fleet",
      details: {
        software_title: "unFLOW SmartClient",
        software_display_name: "Cisco Secure Client - Cloud Management",
        host_display_name: "Test Host",
      },
    });

    render(
      <CanceledSetupExperienceActivityItem activity={activity} tab="past" />
    );

    expect(
      screen.getByText("Cisco Secure Client - Cloud Management")
    ).toBeInTheDocument();
    expect(screen.queryByText("unFLOW SmartClient")).not.toBeInTheDocument();
  });

  it("falls back to the raw name when no display name is set", () => {
    const activity = createMockHostPastActivity({
      type: ActivityType.CanceledSetupExperience,
      actor_full_name: "Fleet",
      details: {
        software_title: "Firefox",
        host_display_name: "Test Host",
      },
    });

    render(
      <CanceledSetupExperienceActivityItem activity={activity} tab="past" />
    );

    expect(screen.getByText("Firefox")).toBeInTheDocument();
  });
});
