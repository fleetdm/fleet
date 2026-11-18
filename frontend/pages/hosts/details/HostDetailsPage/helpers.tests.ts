import createMockHost from "__mocks__/hostMock";
import { MdmEnrollmentStatus } from "interfaces/mdm";
import { HostPlatform } from "interfaces/platform";

import { canShowMyDeviceButton, hasEverEnrolled } from "./helpers";

describe("canShowMyDeviceButton", () => {
  it("returns true when Fleet Desktop is installed and the host is not wiped", () => {
    const host = createMockHost({
      fleet_desktop_version: "1.22.1",
      mdm: { ...createMockHost().mdm, device_status: "unlocked" },
    });
    expect(canShowMyDeviceButton(host, false, true)).toBe(true);
  });

  it("returns true for a locked host that still has Fleet Desktop", () => {
    const host = createMockHost({
      fleet_desktop_version: "1.22.1",
      mdm: { ...createMockHost().mdm, device_status: "locked" },
    });
    expect(canShowMyDeviceButton(host, false, true)).toBe(true);
  });

  // Android and ChromeOS have no My device page, so the link must be hidden
  // regardless of anything else. See #48439.
  it("returns false for Android even with Fleet Desktop reported", () => {
    const host = createMockHost({
      platform: "android",
      fleet_desktop_version: "1.22.1",
      mdm: { ...createMockHost().mdm, device_status: "unlocked" },
    });
    expect(canShowMyDeviceButton(host, false, true)).toBe(false);
  });

  it("returns false for ChromeOS even with Fleet Desktop reported", () => {
    const host = createMockHost({
      platform: "chrome",
      fleet_desktop_version: "1.22.1",
      mdm: { ...createMockHost().mdm, device_status: "unlocked" },
    });
    expect(canShowMyDeviceButton(host, false, true)).toBe(false);
  });

  it("returns false for legacy ChromeOS (CrOS) even with Fleet Desktop reported", () => {
    const host = createMockHost({
      // legacy value predating the HostPlatform union; real hosts can report it
      platform: "CrOS" as HostPlatform,
      fleet_desktop_version: "1.22.1",
      mdm: { ...createMockHost().mdm, device_status: "unlocked" },
    });
    expect(canShowMyDeviceButton(host, false, true)).toBe(false);
  });

  // iOS/iPadOS never run Fleet Desktop; they reach the My device page by host
  // UUID, so the missing fleet_desktop_version must not hide the button.
  it("returns true for iOS without Fleet Desktop", () => {
    const host = createMockHost({
      platform: "ios",
      fleet_desktop_version: null,
      mdm: {
        ...createMockHost().mdm,
        device_status: "unlocked",
        enrollment_status: "On (automatic)",
      },
    });
    expect(canShowMyDeviceButton(host, false, true)).toBe(true);
  });

  it("returns true for iPadOS without Fleet Desktop", () => {
    const host = createMockHost({
      platform: "ipados",
      fleet_desktop_version: null,
      mdm: {
        ...createMockHost().mdm,
        device_status: "unlocked",
        enrollment_status: "On (automatic)",
      },
    });
    expect(canShowMyDeviceButton(host, false, true)).toBe(true);
  });

  it("returns false for a wiped iOS host", () => {
    const host = createMockHost({
      platform: "ios",
      fleet_desktop_version: null,
      mdm: {
        ...createMockHost().mdm,
        device_status: "wiped",
        enrollment_status: "On (automatic)",
      },
    });
    expect(canShowMyDeviceButton(host, false, true)).toBe(false);
  });

  it("returns false when Fleet Desktop is not installed", () => {
    const host = createMockHost({ fleet_desktop_version: null });
    expect(canShowMyDeviceButton(host, false, true)).toBe(false);
  });

  it("returns false when the host has been wiped", () => {
    const host = createMockHost({
      fleet_desktop_version: "1.22.1",
      mdm: { ...createMockHost().mdm, device_status: "wiped" },
    });
    expect(canShowMyDeviceButton(host, false, true)).toBe(false);
  });

  it("returns false when the host has a wipe in flight", () => {
    const host = createMockHost({
      fleet_desktop_version: "1.22.1",
      mdm: {
        ...createMockHost().mdm,
        device_status: "unlocked",
        pending_action: "wipe",
      },
    });
    expect(canShowMyDeviceButton(host, false, true)).toBe(false);
  });

  // Only wipe-related states hide the button. Other transient states leave the
  // end-user page reachable, so the button stays visible.
  it("returns true for non-wipe transient states like clear_passcode", () => {
    const host = createMockHost({
      fleet_desktop_version: "1.22.1",
      mdm: {
        ...createMockHost().mdm,
        device_status: "unlocked",
        pending_action: "clear_passcode",
      },
    });
    expect(canShowMyDeviceButton(host, false, true)).toBe(true);
  });

  it("returns false for ios when fleet desktop SSO is configured", () => {
    const host = createMockHost({
      platform: "ios",
      fleet_desktop_version: "1.22.1",
    });
    expect(canShowMyDeviceButton(host, true, true)).toBe(false);
  });
});

describe("canShowMyDeviceButton for iOS/iPadOS enrollment and tier", () => {
  const iosHost = (enrollmentStatus: MdmEnrollmentStatus | null) =>
    createMockHost({
      platform: "ios",
      fleet_desktop_version: null,
      mdm: {
        ...createMockHost().mdm,
        device_status: "unlocked",
        enrollment_status: enrollmentStatus,
      },
    });

  it.each<MdmEnrollmentStatus | null>(["Pending", "Off", null])(
    "returns false when MDM status is %s",
    (status) => {
      expect(canShowMyDeviceButton(iosHost(status), false, true)).toBe(false);
    }
  );

  it.each<MdmEnrollmentStatus>([
    "On (automatic)",
    "On (manual)",
    "On (personal)",
    "On (manual - personal)",
    "On (company-owned)",
  ])("returns true when MDM status is %s", (status) => {
    expect(canShowMyDeviceButton(iosHost(status), false, true)).toBe(true);
  });

  it("returns false when enrolled but the UUID isn't known yet", () => {
    const host = { ...iosHost("On (automatic)"), uuid: "" };
    expect(canShowMyDeviceButton(host, false, true)).toBe(false);
  });

  it("returns false on Fleet Free even when enrolled", () => {
    expect(canShowMyDeviceButton(iosHost("On (automatic)"), false, false)).toBe(
      false
    );
  });

  it("does not apply the tier or enrollment checks to macOS", () => {
    const host = createMockHost({
      platform: "darwin",
      fleet_desktop_version: "1.22.1",
      mdm: {
        ...createMockHost().mdm,
        device_status: "unlocked",
        enrollment_status: "Off",
      },
    });
    expect(canShowMyDeviceButton(host, false, false)).toBe(true);
  });
});

describe("hasEverEnrolled", () => {
  it("returns true for a host that has checked in", () => {
    expect(
      hasEverEnrolled(
        createMockHost({ last_enrolled_at: "2026-08-21T10:30:00Z" })
      )
    ).toBe(true);
  });

  it("returns false for a pending host holding the never sentinel", () => {
    expect(
      hasEverEnrolled(
        createMockHost({ last_enrolled_at: "2000-01-01T00:00:00Z" })
      )
    ).toBe(false);
  });

  it("returns false when the date is missing", () => {
    expect(hasEverEnrolled(createMockHost({ last_enrolled_at: "" }))).toBe(
      false
    );
  });
});
