import { shouldShowControlsTab } from "./helpers";

describe("shouldShowControlsTab", () => {
  const args = {
    platform: "darwin" as const,
    osVersion: "macOS 15.5",
    enrollmentStatus: "On (manual)" as const,
    hasControls: false,
    isPlatformMdmEnabled: true,
  };

  it("shows the tab for an enrolled MDM host with controls", () => {
    expect(shouldShowControlsTab({ ...args, hasControls: true })).toBe(true);
  });

  it("shows the tab for an enrolled MDM host with no controls, so the empty state renders", () => {
    expect(shouldShowControlsTab(args)).toBe(true);
  });

  it("hides the tab for ChromeOS even when rows were derived", () => {
    expect(
      shouldShowControlsTab({
        ...args,
        platform: "chrome" as const,
        osVersion: "ChromeOS 130",
        hasControls: true,
      })
    ).toBe(false);
  });

  describe("a host with nothing to show", () => {
    it("hides the tab when the platform's MDM is off globally", () => {
      expect(
        shouldShowControlsTab({ ...args, isPlatformMdmEnabled: false })
      ).toBe(false);
    });

    it.each(["Off", null] as const)(
      "hides the tab when the host's enrollment status is %s",
      (enrollmentStatus) => {
        expect(shouldShowControlsTab({ ...args, enrollmentStatus })).toBe(
          false
        );
      }
    );

    // Pending hosts (ABM-synced, not yet enrolled) have no controls applied yet.
    it("hides the tab for a pending MDM host", () => {
      expect(
        shouldShowControlsTab({ ...args, enrollmentStatus: "Pending" })
      ).toBe(false);
    });
  });

  // Rows are the bar the OS settings indicator used before this tab existed, so
  // the tab must show everywhere that did — otherwise unenrolling a host would
  // hide settings it still carries, like Windows disk encryption.
  describe("a host that still carries rows", () => {
    it.each(["Off", null] as const)(
      "shows the tab when enrollment is %s but rows exist",
      (enrollmentStatus) => {
        expect(
          shouldShowControlsTab({
            ...args,
            platform: "windows" as const,
            enrollmentStatus,
            hasControls: true,
            isPlatformMdmEnabled: false,
          })
        ).toBe(true);
      }
    );
  });

  // Fleet only supports disk encryption on Fedora among rhel-likes, so the
  // platform guard has to read os_version, not just the platform.
  describe("rhel-like platforms", () => {
    const rhelArgs = {
      ...args,
      platform: "rhel" as const,
      enrollmentStatus: null,
      isPlatformMdmEnabled: false,
      hasControls: true,
    };

    it("shows the tab for Fedora", () => {
      expect(
        shouldShowControlsTab({ ...rhelArgs, osVersion: "Fedora Linux 40" })
      ).toBe(true);
    });

    it("hides the tab for a non-Fedora rhel host even with rows", () => {
      expect(
        shouldShowControlsTab({
          ...rhelArgs,
          osVersion: "Red Hat Enterprise Linux 9",
        })
      ).toBe(false);
    });
  });

  describe("linux", () => {
    const linuxArgs = {
      ...args,
      platform: "ubuntu" as const,
      osVersion: "Ubuntu 24.04",
      enrollmentStatus: null,
      isPlatformMdmEnabled: false,
    };

    it("shows the tab when disk encryption enforcement derived a row", () => {
      expect(shouldShowControlsTab({ ...linuxArgs, hasControls: true })).toBe(
        true
      );
    });

    it("hides the tab when no controls were derived", () => {
      expect(shouldShowControlsTab(linuxArgs)).toBe(false);
    });
  });

  describe("my device (no per-platform MDM flags available)", () => {
    const deviceArgs = {
      platform: args.platform,
      osVersion: args.osVersion,
      enrollmentStatus: args.enrollmentStatus,
      hasControls: false,
    };

    it("falls back to host enrollment alone", () => {
      expect(shouldShowControlsTab(deviceArgs)).toBe(true);
      expect(
        shouldShowControlsTab({ ...deviceArgs, enrollmentStatus: "Off" })
      ).toBe(false);
    });
  });
});
