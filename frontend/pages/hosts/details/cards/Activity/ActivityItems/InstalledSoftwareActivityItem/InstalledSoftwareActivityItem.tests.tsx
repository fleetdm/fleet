import React from "react";
import { render, screen } from "@testing-library/react";
import { noop } from "lodash";

import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";

import InstalledSoftwareActivityItem from "./InstalledSoftwareActivityItem";

const createInstallActivity = (installSkippedWhenAppOpen?: boolean) =>
  createMockHostPastActivity({
    type: ActivityType.InstalledSoftware,
    actor_full_name: "Fleet",
    fleet_initiated: true,
    details: {
      software_title: "Firefox",
      software_package: "Firefox.pkg",
      host_display_name: "Test Host",
      source: "apps",
      status: "failed_install",
      install_uuid: "uuid-123",
      install_skipped_when_app_open: installSkippedWhenAppOpen,
    },
  });

describe("InstalledSoftwareActivityItem", () => {
  it("renders skipped copy when the app was open", () => {
    render(
      <InstalledSoftwareActivityItem
        activity={createInstallActivity(true)}
        tab="past"
        onShowDetails={noop}
      />
    );

    expect(screen.getByText(/skipped install of/)).toBeInTheDocument();
    expect(screen.getByText("Firefox")).toBeInTheDocument();
    expect(screen.getByText("Test Host")).toBeInTheDocument();
    expect(screen.queryByText(/failed to install/)).not.toBeInTheDocument();
  });

  it("keeps generic failed-install copy when the flag is absent", () => {
    render(
      <InstalledSoftwareActivityItem
        activity={createInstallActivity()}
        tab="past"
        onShowDetails={noop}
      />
    );

    expect(screen.getByText(/failed to install/)).toBeInTheDocument();
    expect(screen.queryByText(/skipped install/)).not.toBeInTheDocument();
  });
});
