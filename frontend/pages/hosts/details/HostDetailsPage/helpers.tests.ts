import createMockHost from "__mocks__/hostMock";

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
});
