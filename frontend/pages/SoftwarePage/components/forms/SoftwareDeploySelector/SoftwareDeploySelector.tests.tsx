import { screen } from "@testing-library/react";
import React from "react";

import {
  getPatchDeadlineSettingsHandler,
  getPatchDeadlineSettingsServerErrorHandler,
} from "test/handlers/patch-deadline-settings-handlers";
import mockServer from "test/mock-server";
import { createCustomRenderer } from "test/test-utils";

import SoftwareDeploySelector, {
  EndUserExperience,
  getPatchPolicyFlags,
  PatchOption,
} from "./SoftwareDeploySelector";

// react-query reads from a shared QueryClient. createCustomRenderer sets up
// its own per-test client, so the Deadline banner's useQuery resolves against
// the MSW handler registered just below.
const render = createCustomRenderer({ withBackendMock: true });

const renderSelector = (
  overrides: Partial<React.ComponentProps<typeof SoftwareDeploySelector>> = {}
) => {
  const props: React.ComponentProps<typeof SoftwareDeploySelector> = {
    forceInstall: false,
    patch: false,
    patchOption: "closed",
    onToggleForceInstall: jest.fn(),
    onTogglePatch: jest.fn(),
    onSelectPatchOption: jest.fn(),
    ...overrides,
  };
  return { ...render(<SoftwareDeploySelector {...props} />), props };
};

describe("SoftwareDeploySelector", () => {
  it("shows Force install without patch options", () => {
    renderSelector({ forceInstall: true });

    expect(
      screen.getByRole("checkbox", { name: "force-install" })
    ).toBeChecked();
    expect(screen.queryByRole("radio")).not.toBeInTheDocument();
  });

  it("shows patch options with Patch when app is closed selected by default", () => {
    renderSelector({ patch: true });

    expect(screen.getByRole("checkbox", { name: "patch" })).toBeChecked();
    expect(
      screen.getByRole("radio", { name: "Patch when app is closed" })
    ).toBeChecked();
  });

  it("supports Force install and Patch together", () => {
    renderSelector({ forceInstall: true, patch: true });

    expect(
      screen.getByRole("checkbox", { name: "force-install" })
    ).toBeChecked();
    expect(screen.getByRole("checkbox", { name: "patch" })).toBeChecked();
    expect(screen.getAllByRole("radio")).toHaveLength(3);
  });

  it("shows the Deadline banner by default on macOS Force patch", async () => {
    mockServer.use(getPatchDeadlineSettingsHandler());
    renderSelector({
      patch: true,
      patchOption: "force",
      platform: "darwin",
      endUserExperience: "deadline",
      onSelectEndUserExperience: jest.fn(),
      fleetId: 1,
    });

    // Wait for the useQuery result to flow into the banner — the initial
    // render shows "---" placeholders, so wait on the resolved weekday.
    expect(
      await screen.findByText("Tuesday", { exact: false })
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        /Patch is done silently when app is closed or forced at/i
      )
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /Edit deadline/i })
    ).toHaveAttribute(
      "href",
      expect.stringContaining(
        "/software/library/settings/patch-deadline?fleet_id=1"
      )
    );
  });

  it("renders Deadline banner placeholders when the settings endpoint errors", async () => {
    mockServer.use(getPatchDeadlineSettingsServerErrorHandler);
    renderSelector({
      patch: true,
      patchOption: "force",
      platform: "darwin",
      endUserExperience: "deadline",
      onSelectEndUserExperience: jest.fn(),
      fleetId: 1,
    });

    // Error state still shows the banner shell with the dashes placeholder so
    // the "Edit deadline" link stays reachable while the settings page loads.
    expect(
      await screen.findByText(
        /Patch is done silently when app is closed or forced at/i
      )
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /Edit deadline/i })
    ).toBeInTheDocument();
  });

  it("shows the Notify before patching banner when selected on macOS", () => {
    renderSelector({
      patch: true,
      patchOption: "force",
      platform: "darwin",
      endUserExperience: "notify",
      onSelectEndUserExperience: jest.fn(),
    });

    expect(
      screen.getByText(/End user is notified when the policy fails/)
    ).toBeInTheDocument();
    expect(screen.getByText(/1 hour/)).toBeInTheDocument();
  });

  it("shows the Patch immediately banner when selected on macOS", () => {
    renderSelector({
      patch: true,
      patchOption: "force",
      platform: "darwin",
      endUserExperience: "immediate",
      onSelectEndUserExperience: jest.fn(),
    });

    expect(
      screen.getByText(
        /End user is not notified\. Patch is forced as soon as policy fails\./
      )
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /End user experience/i })
    ).toHaveAttribute(
      "href",
      "https://fleetdm.com/learn-more-about/patching-end-user-experience"
    );
  });

  it("shows the Windows coming-soon banner and no dropdown on Windows", () => {
    renderSelector({
      patch: true,
      patchOption: "force",
      platform: "windows",
      endUserExperience: "deadline",
      onSelectEndUserExperience: jest.fn(),
    });

    expect(
      screen.getByText(/Notifications for Windows are coming soon\./)
    ).toBeInTheDocument();
    expect(screen.queryByText("End user experience")).not.toBeInTheDocument();
  });

  it("hides the End user experience dropdown for non-Force patch options", () => {
    renderSelector({
      patch: true,
      patchOption: "closed",
      platform: "darwin",
      endUserExperience: "deadline",
      onSelectEndUserExperience: jest.fn(),
    });

    expect(screen.queryByText("End user experience")).not.toBeInTheDocument();
  });

  describe("getPatchPolicyFlags", () => {
    it("returns force+deadline flags for Deadline", () => {
      const flags = getPatchPolicyFlags(
        "force",
        "deadline" as EndUserExperience
      );
      expect(flags.patch_deadline).toBe(true);
      expect(flags.notify_before_patching).toBe(false);
      expect(flags.patch_when_closed).toBe(false);
      expect(flags.continuous_automations_enabled).toBe(true);
    });

    it("returns force+notify flags for Notify before patching", () => {
      const flags = getPatchPolicyFlags("force", "notify" as EndUserExperience);
      expect(flags.notify_before_patching).toBe(true);
      expect(flags.patch_deadline).toBe(false);
      expect(flags.patch_when_closed).toBe(false);
      expect(flags.continuous_automations_enabled).toBe(true);
    });

    it("returns all false for Patch immediately", () => {
      const flags = getPatchPolicyFlags("force", "immediate");
      expect(flags.notify_before_patching).toBe(false);
      expect(flags.patch_deadline).toBe(false);
      expect(flags.patch_when_closed).toBe(false);
      expect(flags.continuous_automations_enabled).toBe(false);
    });

    it("returns patch_when_closed for closed regardless of experience", () => {
      const flags = getPatchPolicyFlags("closed");
      expect(flags.patch_when_closed).toBe(true);
      expect(flags.notify_before_patching).toBe(false);
      expect(flags.patch_deadline).toBe(false);
      expect(flags.continuous_automations_enabled).toBe(true);
    });

    it("defaults to deadline when no end-user experience is passed", () => {
      // Catches the default argument regressing to "immediate" and silently
      // flipping force patches off deadline.
      const flags = getPatchPolicyFlags("force");
      expect(flags.patch_deadline).toBe(true);
    });
  });

  it("shows the pre-install query override notice in the Deploy modal", () => {
    renderSelector({ patch: true, showPatchWhenClosedNotice: true });

    expect(
      screen.getByText(/overrides the pre-install query \(advanced option\)/)
    ).toBeInTheDocument();
  });

  it("reports the selected patch option", async () => {
    const onSelectPatchOption = jest.fn<void, [PatchOption]>();
    const { getByRole } = renderSelector({
      patch: true,
      onSelectPatchOption,
    });

    getByRole("radio", { name: "Force patch" }).click();
    expect(onSelectPatchOption).toHaveBeenCalledWith("force");
  });
});
