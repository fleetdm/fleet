import React from "react";
import { render, screen } from "@testing-library/react";
import { noop } from "lodash";

import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";

import InstalledSoftwareActivityItem from "./InstalledSoftwareActivityItem";

const createInstallActivity = (skippedInstall?: boolean) =>
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
      skipped_install: skippedInstall,
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

  it("attributes VPP auto-update installs to Fleet even when the payload carries a stale actor", () => {
    const activity = createMockHostPastActivity({
      type: ActivityType.InstalledAppStoreApp,
      actor_full_name: "Some Admin",
      fleet_initiated: false,
      details: {
        software_title: "Google Meet",
        host_display_name: "iPad",
        source: "ipados_apps",
        status: "installed",
        command_uuid: "cmd-1",
        from_auto_update: true,
      },
    });

    render(
      <InstalledSoftwareActivityItem
        activity={activity}
        tab="past"
        onShowDetails={noop}
      />
    );

    expect(screen.getByText("Fleet")).toBeInTheDocument();
    expect(screen.queryByText("Some Admin")).not.toBeInTheDocument();
  });
});
