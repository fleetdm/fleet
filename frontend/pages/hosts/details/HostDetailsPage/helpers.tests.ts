import createMockHost from "__mocks__/hostMock";

import { canShowMyDeviceButton, hasEverEnrolled } from "./helpers";

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
