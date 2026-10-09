import { render, screen } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType, IHostPastActivity } from "interfaces/activity";

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

  it("hides the Show details button when hideShowDetails is true", () => {
    render(
      <InstalledSoftwareActivityItem
        activity={createInstallActivity()}
        tab="past"
        onShowDetails={noop}
        hideShowDetails
      />
    );

    expect(
      screen.queryByRole("button", { name: /show info/i })
    ).not.toBeInTheDocument();
  });

  describe("version suffix", () => {
    const createVppActivity = (
      detailsOverrides: Partial<
        IHostPastActivity["details"] & Record<string, unknown>
      >
    ) =>
      createMockHostPastActivity({
        type: ActivityType.InstalledAppStoreApp,
        actor_full_name: "Some Admin",
        fleet_initiated: false,
        details: {
          software_title: "Google Meet",
          host_display_name: "iPad",
          source: "ipados_apps",
          status: "installed",
          command_uuid: "cmd-1",
          ...detailsOverrides,
        },
      });

    it("appends the mobile version suffix in the normal branch", () => {
      render(
        <InstalledSoftwareActivityItem
          activity={createVppActivity({
            host_platform: "ipados",
            version_name: "Default version",
          })}
          tab="past"
          onShowDetails={noop}
        />
      );

      expect(screen.getByText("Google Meet")).toBeInTheDocument();
      expect(screen.getByText(/\(Default version\)/)).toBeInTheDocument();
    });

    it("appends the mobile version suffix in the skipped branch", () => {
      render(
        <InstalledSoftwareActivityItem
          activity={createVppActivity({
            status: "failed_install",
            skipped_install: true,
            host_platform: "ios",
            version_name: "Default version",
          })}
          tab="past"
          onShowDetails={noop}
        />
      );

      expect(screen.getByText("Google Meet")).toBeInTheDocument();
      expect(screen.getByText(/\(Default version\)/)).toBeInTheDocument();
    });

    it("appends the mobile version suffix in the self-service branch", () => {
      render(
        <InstalledSoftwareActivityItem
          activity={createVppActivity({
            self_service: true,
            host_platform: "android",
            version_name: "Default version",
          })}
          tab="past"
          onShowDetails={noop}
        />
      );

      expect(screen.getByText("Google Meet")).toBeInTheDocument();
      expect(screen.getByText(/\(Default version\)/)).toBeInTheDocument();
      expect(screen.getByText(/\(self service\)/)).toBeInTheDocument();
    });

    it("omits the version suffix on macOS even when version_name is set", () => {
      render(
        <InstalledSoftwareActivityItem
          activity={createVppActivity({
            source: "apps",
            host_platform: "darwin",
            version_name: "Default version",
          })}
          tab="past"
          onShowDetails={noop}
        />
      );

      expect(screen.getByText("Google Meet")).toBeInTheDocument();
      expect(screen.queryByText(/\(Default version\)/)).not.toBeInTheDocument();
    });
  });
});
