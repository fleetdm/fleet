import React from "react";
import { render, screen } from "@testing-library/react";

import SoftwareDeploySelector, {
  EndUserExperience,
  getPatchPolicyFlags,
  PatchOption,
} from "./SoftwareDeploySelector";

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

  it("shows the Patch immediately banner by default on macOS Force patch", () => {
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

  it("shows the Windows coming-soon banner and no dropdown on Windows", () => {
    renderSelector({
      patch: true,
      patchOption: "force",
      platform: "windows",
      endUserExperience: "immediate",
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
      endUserExperience: "immediate",
      onSelectEndUserExperience: jest.fn(),
    });

    expect(screen.queryByText("End user experience")).not.toBeInTheDocument();
  });

  describe("getPatchPolicyFlags", () => {
    it("returns force+notify flags for Notify before patching on macOS", () => {
      const flags = getPatchPolicyFlags(
        "force",
        "notify" as EndUserExperience,
        "darwin"
      );
      expect(flags.notify_before_patching).toBe(true);
      expect(flags.patch_when_closed).toBe(false);
      expect(flags.continuous_automations_enabled).toBe(true);
    });

    it("returns all false for Patch immediately", () => {
      const flags = getPatchPolicyFlags("force", "immediate", "darwin");
      expect(flags.notify_before_patching).toBe(false);
      expect(flags.patch_when_closed).toBe(false);
      expect(flags.continuous_automations_enabled).toBe(false);
    });

    it("returns patch_when_closed for closed regardless of experience", () => {
      const flags = getPatchPolicyFlags("closed");
      expect(flags.patch_when_closed).toBe(true);
      expect(flags.notify_before_patching).toBe(false);
      expect(flags.continuous_automations_enabled).toBe(true);
    });

    it("never sets notify_before_patching for Windows even when state says notify", () => {
      const flags = getPatchPolicyFlags(
        "force",
        "notify" as EndUserExperience,
        "windows"
      );
      expect(flags.notify_before_patching).toBe(false);
      expect(flags.continuous_automations_enabled).toBe(false);
    });

    it("never sets notify_before_patching for an unknown platform", () => {
      const flags = getPatchPolicyFlags(
        "force",
        "notify" as EndUserExperience,
        undefined
      );
      expect(flags.notify_before_patching).toBe(false);
      expect(flags.continuous_automations_enabled).toBe(false);
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
