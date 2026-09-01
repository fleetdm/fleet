import createMockHost from "__mocks__/hostMock";
import { HostPlatform } from "interfaces/platform";

import { canShowMyDeviceButton } from "./helpers";

describe("canShowMyDeviceButton", () => {
  it("returns true when Fleet Desktop is installed and the host is not wiped", () => {
    const host = createMockHost({
      fleet_desktop_version: "1.22.1",
      mdm: { ...createMockHost().mdm, device_status: "unlocked" },
    });
    expect(canShowMyDeviceButton(host)).toBe(true);
  });

  it("returns true for a locked host that still has Fleet Desktop", () => {
    const host = createMockHost({
      fleet_desktop_version: "1.22.1",
      mdm: { ...createMockHost().mdm, device_status: "locked" },
    });
    expect(canShowMyDeviceButton(host)).toBe(true);
  });

  // Android and ChromeOS have no My device page, so the link must be hidden
  // regardless of anything else. See #48439.
  it("returns false for Android even with Fleet Desktop reported", () => {
    const host = createMockHost({
      platform: "android",
      fleet_desktop_version: "1.22.1",
      mdm: { ...createMockHost().mdm, device_status: "unlocked" },
    });
    expect(canShowMyDeviceButton(host)).toBe(false);
  });

  it("returns false for ChromeOS even with Fleet Desktop reported", () => {
    const host = createMockHost({
      platform: "chrome",
      fleet_desktop_version: "1.22.1",
      mdm: { ...createMockHost().mdm, device_status: "unlocked" },
    });
    expect(canShowMyDeviceButton(host)).toBe(false);
  });

  it("returns false for legacy ChromeOS (CrOS) even with Fleet Desktop reported", () => {
    const host = createMockHost({
      // legacy value predating the HostPlatform union; real hosts can report it
      platform: "CrOS" as HostPlatform,
      fleet_desktop_version: "1.22.1",
      mdm: { ...createMockHost().mdm, device_status: "unlocked" },
    });
    expect(canShowMyDeviceButton(host)).toBe(false);
  });

  it("returns false when Fleet Desktop is not installed", () => {
    const host = createMockHost({ fleet_desktop_version: null });
    expect(canShowMyDeviceButton(host)).toBe(false);
  });

  it("returns false when the host has been wiped", () => {
    const host = createMockHost({
      fleet_desktop_version: "1.22.1",
      mdm: { ...createMockHost().mdm, device_status: "wiped" },
    });
    expect(canShowMyDeviceButton(host)).toBe(false);
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
    expect(canShowMyDeviceButton(host)).toBe(false);
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
    expect(canShowMyDeviceButton(host)).toBe(true);
  });
});
