import PATHS from "router/paths";

import {
  softwareSubNav,
  premiumSoftwareSubNav,
  getTabIndex,
  getOSTabSortHeader,
} from "./SoftwarePage";

// These are not exported by default — we'll test the logic via the exported
// nav arrays and getTabIndex. If they aren't exported yet, see note below.

describe("SoftwarePage tab configuration", () => {
  describe("softwareSubNav (free tier)", () => {
    it("includes Inventory, OS, and Vulnerabilities tabs", () => {
      const names = softwareSubNav.map((item) => item.name);
      expect(names).toEqual(["Inventory", "OS", "Vulnerabilities"]);
    });

    it("does not include Library tab", () => {
      const names = softwareSubNav.map((item) => item.name);
      expect(names).not.toContain("Library");
    });

    it("points Inventory to SOFTWARE_INVENTORY path", () => {
      const inventory = softwareSubNav.find(
        (item) => item.name === "Inventory"
      );
      expect(inventory?.pathname).toBe(PATHS.SOFTWARE_INVENTORY);
    });
  });

  describe("premiumSoftwareSubNav (premium tier)", () => {
    it("includes Inventory, OS, Vulnerabilities, and Library tabs", () => {
      const names = premiumSoftwareSubNav.map((item) => item.name);
      expect(names).toEqual(["Inventory", "OS", "Vulnerabilities", "Library"]);
    });

    it("points Library to SOFTWARE_LIBRARY path", () => {
      const library = premiumSoftwareSubNav.find(
        (item) => item.name === "Library"
      );
      expect(library?.pathname).toBe(PATHS.SOFTWARE_LIBRARY);
    });
  });

  describe("getTabIndex", () => {
    it("returns the Inventory tab index for the inventory path", () => {
      expect(getTabIndex(PATHS.SOFTWARE_INVENTORY, premiumSoftwareSubNav)).toBe(
        0
      );
    });

    it("returns the Inventory tab index for the versions path", () => {
      expect(getTabIndex(PATHS.SOFTWARE_VERSIONS, premiumSoftwareSubNav)).toBe(
        0
      );
    });

    it("returns the OS tab index for the OS path", () => {
      expect(getTabIndex(PATHS.SOFTWARE_OS, premiumSoftwareSubNav)).toBe(1);
    });

    it("returns the Vulnerabilities tab index for the vulnerabilities path", () => {
      expect(
        getTabIndex(PATHS.SOFTWARE_VULNERABILITIES, premiumSoftwareSubNav)
      ).toBe(2);
    });

    it("returns the Library tab index for the library path", () => {
      expect(getTabIndex(PATHS.SOFTWARE_LIBRARY, premiumSoftwareSubNav)).toBe(
        3
      );
    });

    it("returns -1 for an unknown path", () => {
      expect(getTabIndex("/software/unknown", premiumSoftwareSubNav)).toBe(-1);
    });
  });

  describe("getOSTabSortHeader", () => {
    it("defaults to version once a single platform is selected on the OS tab", () => {
      expect(getOSTabSortHeader(PATHS.SOFTWARE_OS, "darwin", undefined)).toBe(
        "version"
      );
    });

    it("defaults to host count on the OS tab's 'All platforms' view", () => {
      expect(getOSTabSortHeader(PATHS.SOFTWARE_OS, "all", undefined)).toBe(
        "hosts_count"
      );
    });

    it("ignores a crafted/stale order_key=version on 'All platforms', instead of sending a nonsensical cross-platform version sort to the API", () => {
      // Comparing OS versions across platforms isn't meaningful, and the
      // Version column is unclickable on this view — but this page is
      // server-driven, so a URL with both params together (typed by hand,
      // bookmarked, or restored via browser back/forward) would otherwise
      // still reach the API as a real order_key=version request.
      expect(getOSTabSortHeader(PATHS.SOFTWARE_OS, "all", "version")).toBe(
        "hosts_count"
      );
    });

    it("still honors an explicit order_key on a single platform, or on other tabs", () => {
      expect(getOSTabSortHeader(PATHS.SOFTWARE_OS, "darwin", "name")).toBe(
        "name"
      );
      expect(
        getOSTabSortHeader(PATHS.SOFTWARE_INVENTORY, "all", "version")
      ).toBe("version");
    });
  });
});
