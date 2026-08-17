import { shouldShowControlsTab } from "./helpers";

describe("shouldShowControlsTab", () => {
  const args = {
    platform: "darwin",
    enrollmentStatus: "On (manual)" as const,
    hasControls: true,
    isPlatformMdmEnabled: true,
  };

  it("shows the tab for an enrolled MDM host", () => {
    expect(shouldShowControlsTab(args)).toBe(true);
  });

  it("shows the tab for an enrolled MDM host with no controls, so the empty state renders", () => {
    expect(shouldShowControlsTab({ ...args, hasControls: false })).toBe(true);
  });

  it("hides the tab for ChromeOS", () => {
    expect(shouldShowControlsTab({ ...args, platform: "chrome" })).toBe(false);
  });

  it("hides the tab when the platform's MDM is off globally", () => {
    expect(
      shouldShowControlsTab({ ...args, isPlatformMdmEnabled: false })
    ).toBe(false);
  });

  it.each(["Off", null] as const)(
    "hides the tab when the host's enrollment status is %s",
    (enrollmentStatus) => {
      expect(shouldShowControlsTab({ ...args, enrollmentStatus })).toBe(false);
    }
  );

  // Pending hosts (ABM-synced, not yet enrolled) have no controls applied yet.
  it("hides the tab for a pending MDM host", () => {
    expect(
      shouldShowControlsTab({ ...args, enrollmentStatus: "Pending" })
    ).toBe(false);
  });

  describe("linux", () => {
    const linuxArgs = {
      ...args,
      platform: "ubuntu",
      enrollmentStatus: null,
      isPlatformMdmEnabled: false,
    };

    it("shows the tab when disk encryption enforcement derived a row", () => {
      expect(shouldShowControlsTab(linuxArgs)).toBe(true);
    });

    it("hides the tab when no controls were derived", () => {
      expect(shouldShowControlsTab({ ...linuxArgs, hasControls: false })).toBe(
        false
      );
    });
  });

  describe("my device (no per-platform MDM flags available)", () => {
    const deviceArgs = {
      platform: args.platform,
      enrollmentStatus: args.enrollmentStatus,
      hasControls: args.hasControls,
    };

    it("falls back to host enrollment alone", () => {
      expect(shouldShowControlsTab(deviceArgs)).toBe(true);
      expect(
        shouldShowControlsTab({ ...deviceArgs, enrollmentStatus: "Off" })
      ).toBe(false);
    });
  });
});
