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

  // Hiding the tab would leave these hosts with no explanation at all: the
  // "Turn on MDM" banner on both Host details and My device is macOS-only.
  describe("an unenrolled host whose platform MDM is on", () => {
    it.each(["Off", null, "Pending"] as const)(
      "shows the tab when enrollment is %s",
      (enrollmentStatus) => {
        expect(
          shouldShowControlsTab({
            ...args,
            platform: "windows" as const,
            osVersion: "Windows 11",
            enrollmentStatus,
          })
        ).toBe(true);
      }
    );
  });

  describe("a host with nothing to show", () => {
    const nothingToShow = {
      ...args,
      enrollmentStatus: null,
      isPlatformMdmEnabled: false,
    };

    it("hides the tab when the platform's MDM is off and the host isn't enrolled", () => {
      expect(shouldShowControlsTab(nothingToShow)).toBe(false);
    });

    // Turning MDM off globally can leave a host still reading as enrolled.
    it("shows the tab when the host still reads as enrolled", () => {
      expect(
        shouldShowControlsTab({
          ...nothingToShow,
          enrollmentStatus: "On (manual)",
        })
      ).toBe(true);
    });
  });

  // Rows are the bar the OS settings indicator used before this tab existed, so
  // the tab must show everywhere that did — otherwise unenrolling a host would
  // hide settings it still carries, like Windows disk encryption.
  it("shows the tab whenever rows exist, whatever the MDM state", () => {
    expect(
      shouldShowControlsTab({
        ...args,
        platform: "windows" as const,
        osVersion: "Windows 11",
        enrollmentStatus: null,
        isPlatformMdmEnabled: false,
        hasControls: true,
      })
    ).toBe(true);
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

    // My device omits isPlatformMdmEnabled, which defaults to true — without
    // the Linux guard that default would show the tab there but not on Host
    // details, and claim the host "isn't enrolled in MDM".
    it("hides the tab on My device, where the global MDM flag is unavailable", () => {
      expect(
        shouldShowControlsTab({
          platform: linuxArgs.platform,
          osVersion: linuxArgs.osVersion,
          enrollmentStatus: linuxArgs.enrollmentStatus,
          hasControls: false,
        })
      ).toBe(false);
    });
  });

  // My device can't read per-platform MDM flags, so the tab shows on any
  // OS-settings platform there.
  describe("my device", () => {
    const deviceArgs = {
      platform: args.platform,
      osVersion: args.osVersion,
      enrollmentStatus: null,
      hasControls: false,
    };

    it("shows the tab without a global MDM flag to check", () => {
      expect(shouldShowControlsTab(deviceArgs)).toBe(true);
    });
  });
});
