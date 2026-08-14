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

  it("renders the software's display name instead of its raw name when set", () => {
    const activity = createMockHostPastActivity({
      type: ActivityType.InstalledSoftware,
      actor_full_name: "Fleet",
      fleet_initiated: true,
      details: {
        software_title: "unFLOW SmartClient",
        software_display_name: "Cisco Secure Client - Cloud Management",
        software_package: "cisco.pkg",
        host_display_name: "Test Host",
        source: "apps",
        status: "installed",
        install_uuid: "uuid-456",
      },
    });

    render(
      <InstalledSoftwareActivityItem
        activity={activity}
        tab="past"
        onShowDetails={noop}
      />
    );

    expect(
      screen.getByText("Cisco Secure Client - Cloud Management")
    ).toBeInTheDocument();
    expect(screen.queryByText("unFLOW SmartClient")).not.toBeInTheDocument();
  });
});
