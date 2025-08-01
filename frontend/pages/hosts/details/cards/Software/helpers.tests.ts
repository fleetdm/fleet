import {
  createMockHostSoftware,
  createMockHostSoftwarePackage,
} from "__mocks__/hostMock";
import { compareVersions, getUiStatus, getSoftwareSubheader } from "./helpers";

describe("compareVersions", () => {
  it("correctly compares patch increments", () => {
    expect(compareVersions("1.0.0", "1.0.1")).toBe(-1);
    expect(compareVersions("1.0.1", "1.0.0")).toBe(1);
    expect(compareVersions("1.0.0", "1.0.0")).toBe(0);
  });

  it("handles pre-release after stable", () => {
    expect(compareVersions("1.0.0", "1.0.0-rc.1")).toBe(1);
    expect(compareVersions("1.0.0-rc.1", "1.0.0")).toBe(-1);
  });

  it("orders pre-release tags correctly", () => {
    expect(compareVersions("1.0.0-alpha", "1.0.0-beta")).toBe(-1);
    expect(compareVersions("1.0.0-beta", "1.0.0-rc")).toBe(-1);
    expect(compareVersions("1.0.0-rc", "1.0.0")).toBe(-1);
    expect(compareVersions("1.0.0-alpha", "1.0.0-rc")).toBe(-1);
  });

  it("orders pre-release tags correctly against patch increments", () => {
    expect(compareVersions("1.0", "1.2-beta")).toBe(-1);
  });

  it("compares numeric suffixes after pre-release tags", () => {
    expect(compareVersions("1.0.0-alpha.1", "1.0.0-alpha.2")).toBe(-1);
    expect(compareVersions("1.0.0-rc.1", "1.0.0-rc.2")).toBe(-1);
    expect(compareVersions("1.0.0-rc.4", "1.0.0-rc.3")).toBe(1);
  });

  it("handles alphanumeric suffixes", () => {
    expect(compareVersions("1.0.0a", "1.0.0b")).toBe(-1);
    expect(compareVersions("1.0.0b", "1.0.0a")).toBe(1);
  });

  it("treats shorter and longer versions as equal if trailing zeros", () => {
    expect(compareVersions("1.0", "1.0.0")).toBe(0);
    expect(compareVersions("1.0.0", "1.0")).toBe(0);
    expect(compareVersions("1.0.0", "1.0.0.0")).toBe(0);
  });

  it("compares numeric segments correctly", () => {
    expect(compareVersions("1.0.9", "1.0.10")).toBe(-1);
    expect(compareVersions("1.0.10", "1.0.9")).toBe(1);
  });

  it("handles date-based versioning", () => {
    expect(compareVersions("2023.12.31", "2024.01.01")).toBe(-1);
    expect(compareVersions("2024.01.01", "2023.12.31")).toBe(1);
  });

  it('handles leading "v" in version strings', () => {
    expect(compareVersions("v1.0.0", "v2.0.0")).toBe(-1);
    expect(compareVersions("v2.0.0", "v1.0.0")).toBe(1);
  });

  it("treats build metadata as equal (if supported)", () => {
    expect(compareVersions("1.0.0+20130313144700", "1.0.0")).toBe(0);
  });

  it("is case-insensitive for pre-release tags", () => {
    expect(compareVersions("1.0.0-Alpha", "1.0.0-alpha")).toBe(0);
    expect(compareVersions("1.0.0-BETA", "1.0.0-beta")).toBe(0);
  });

  it("ignores leading zeros in numeric segments", () => {
    expect(compareVersions("1.01.0", "1.1.0")).toBe(0);
    expect(compareVersions("01.1.0", "1.1.0")).toBe(0);
  });

  it("compares build number in parentheses", () => {
    expect(compareVersions("6.1.11 (39163)", "6.1.11 (30000)")).toBe(1);
  });
});

describe("getUiStatus", () => {
  it("returns 'failed_install_update_available' when failed_install and update available", () => {
    const sw = createMockHostSoftware({
      status: "failed_install",
      software_package: createMockHostSoftwarePackage({ version: "2.0.0" }), // version higher than installed
    });
    expect(getUiStatus(sw, true)).toBe("failed_install_update_available");
  });

  it("returns 'failed_install' when failed_install and no update available", () => {
    const sw = createMockHostSoftware({
      status: "failed_install",
      // version equal to installed version
    });
    expect(getUiStatus(sw, true)).toBe("failed_install");
  });

  it("returns 'failed_uninstall_update_available' when failed_uninstall and update available", () => {
    const sw = createMockHostSoftware({
      status: "failed_uninstall",
      software_package: createMockHostSoftwarePackage({ version: "2.0.0" }), // version higher than installed
    });
    expect(getUiStatus(sw, true)).toBe("failed_uninstall_update_available");
  });

  it("returns 'failed_uninstall' when failed_uninstall and no update available", () => {
    const sw = createMockHostSoftware({
      status: "failed_uninstall",
      // version equal to installed version
    });
    expect(getUiStatus(sw, true)).toBe("failed_uninstall");
  });

  it("returns 'updating' if pending_install and update is available, host online", () => {
    const sw = createMockHostSoftware({
      status: "pending_install",
      software_package: createMockHostSoftwarePackage({ version: "2.0.0" }), // version higher than installed
    });
    expect(getUiStatus(sw, true)).toBe("updating");
  });

  it("returns 'pending_update' if pending_install and update is available, host offline", () => {
    const sw = createMockHostSoftware({
      status: "pending_install",
      software_package: createMockHostSoftwarePackage({ version: "2.0.0" }), // version higher than installed
    });
    expect(getUiStatus(sw, false)).toBe("pending_update");
  });

  it("returns 'installing' if pending_install and reinstalling, host online", () => {
    const sw = createMockHostSoftware({
      status: "pending_install",
    });
    expect(getUiStatus(sw, true)).toBe("installing");
  });

  it("returns 'pending_install' if pending_install and reinstalling, host offline", () => {
    const sw = createMockHostSoftware({
      status: "pending_install",
    });
    expect(getUiStatus(sw, false)).toBe("pending_install");
  });

  it("returns 'installing' if pending_install and nothing installed, host online", () => {
    const sw = createMockHostSoftware({
      status: "pending_install",
      installed_versions: [],
    });
    expect(getUiStatus(sw, true)).toBe("installing");
  });

  it("returns 'pending_install' if pending_install and nothing installed, host offline", () => {
    const sw = createMockHostSoftware({
      status: "pending_install",
      installed_versions: [],
    });
    expect(getUiStatus(sw, false)).toBe("pending_install");
  });

  it("returns 'uninstalling' if pending_uninstall and host online", () => {
    const sw = createMockHostSoftware({
      status: "pending_uninstall",
    });
    expect(getUiStatus(sw, true)).toBe("uninstalling");
  });

  it("returns 'pending_uninstall' if pending_uninstall and host offline", () => {
    const sw = createMockHostSoftware({
      status: "pending_uninstall",
    });
    expect(getUiStatus(sw, false)).toBe("pending_uninstall");
  });

  it("returns 'update_available' if status not pending and update available", () => {
    const sw = createMockHostSoftware({
      status: "installed",
      software_package: createMockHostSoftwarePackage({ version: "2.0.0" }), // version higher than installed
    });
    expect(getUiStatus(sw, true)).toBe("update_available");
  });

  // Tarball packages (tgz_packages) are not tracked in software inventory
  // so they should return 'installed' if their status is installed.
  it("returns 'installed' for tgz_packages with status installed", () => {
    const sw = createMockHostSoftware({
      status: "installed",
      source: "tgz_packages",
    });
    expect(getUiStatus(sw, true)).toBe("installed");
  });

  it("returns 'installed' for regular package, installed and versions match", () => {
    const sw = createMockHostSoftware({
      status: "installed",
    });
    expect(getUiStatus(sw, true)).toBe("installed");
  });

  it("returns 'installed' for regular package, installed version higher than library version", () => {
    const sw = createMockHostSoftware({
      status: "installed",
      software_package: createMockHostSoftwarePackage({ version: "0.1.0" }), // version lower than installed
    });
    expect(getUiStatus(sw, true)).toBe("installed");
  });

  it("returns 'uninstalled' if no conditions match", () => {
    const sw = createMockHostSoftware({
      status: null,
      installed_versions: [],
    });
    expect(getUiStatus(sw, true)).toBe("uninstalled");
  });
});

describe("getSoftwareSubheader", () => {
  test("iOS device, MDM status 'On (personal)', my device page", () => {
    const result = getSoftwareSubheader({
      platform: "ios",
      hostMdmEnrollmentStatus: "On (personal)",
      isMyDevicePage: true,
    });
    expect(result).toBe(
      "Software installed on your work profile (Managed Apple Account)."
    );
  });

  test("iOS device, MDM status 'On (personal)', NOT my device page", () => {
    const result = getSoftwareSubheader({
      platform: "ios",
      hostMdmEnrollmentStatus: "On (personal)",
      isMyDevicePage: false,
    });
    expect(result).toBe(
      "Software installed on work profile (Managed Apple Account)."
    );
  });

  test("iOS device, MDM status 'On (manual)', my device page", () => {
    const result = getSoftwareSubheader({
      platform: "ios",
      hostMdmEnrollmentStatus: "On (manual)",
      isMyDevicePage: true,
    });
    expect(result).toBe(
      "Software installed on your device. Built-in apps (ex. Calculator) aren't included."
    );
  });

  test("iOS device, MDM status 'On (manual)', NOT my device page", () => {
    const result = getSoftwareSubheader({
      platform: "ios",
      hostMdmEnrollmentStatus: "On (manual)",
      isMyDevicePage: false,
    });
    expect(result).toBe(
      "Software installed on this host. Built-in apps (ex. Calculator) aren't included."
    );
  });

  test("iOS device, MDM status not special, my device page", () => {
    const result = getSoftwareSubheader({
      platform: "ios",
      hostMdmEnrollmentStatus: "Off",
      isMyDevicePage: true,
    });
    expect(result).toBe("Software installed on your device");
  });

  test("iOS device, MDM status not special, NOT my device page", () => {
    const result = getSoftwareSubheader({
      platform: "ios",
      hostMdmEnrollmentStatus: "Off",
      isMyDevicePage: false,
    });
    expect(result).toBe("Software installed on this host");
  });

  test("default (NOT iOS device) my device page", () => {
    const result = getSoftwareSubheader({
      platform: "windows",
      hostMdmEnrollmentStatus: "Off",
      isMyDevicePage: true,
    });
    expect(result).toBe("Software installed on your device");
  });

  test("default (NOT iOS device) NOT my device page", () => {
    const result = getSoftwareSubheader({
      platform: "windows",
      hostMdmEnrollmentStatus: "Off",
      isMyDevicePage: false,
    });
    expect(result).toBe("Software installed on this host");
  });
});
