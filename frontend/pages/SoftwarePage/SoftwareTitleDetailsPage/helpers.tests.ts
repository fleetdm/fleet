import { ISoftwareTitleDetails } from "interfaces/software";
import { buildLibraryVersionRows, getInstallerCardInfo } from "./helpers";

const v = (id: number, version: string) => ({
  id,
  version,
  filename: `installer-${version}.pkg`,
  uploaded_at: "2026-01-01T00:00:00Z",
});

describe("SoftwareTitleDetailsPage helpers", () => {
  describe("buildLibraryVersionRows", () => {
    it("renders a single active 'Latest' row when there is no cached-version list", () => {
      expect(
        buildLibraryVersionRows({
          fleetMaintainedVersions: null,
          activeVersion: "1.2.3",
          pinnedVersion: null,
          addedTimestamp: "2026-02-02T00:00:00Z",
        })
      ).toEqual([
        {
          id: -1,
          version: "1.2.3",
          uploaded_at: "2026-02-02T00:00:00Z",
          isActive: true,
          badgeState: "latest",
        },
      ]);
    });

    it("marks the active-version row active with a 'latest' badge when unpinned", () => {
      const rows = buildLibraryVersionRows({
        fleetMaintainedVersions: [
          v(1, "149.0.7827.54"),
          v(2, "148.0.7778.179"),
        ],
        activeVersion: "149.0.7827.54",
        pinnedVersion: null,
        addedTimestamp: "x",
      });
      expect(rows.map((r) => [r.version, r.isActive, r.badgeState])).toEqual([
        ["149.0.7827.54", true, "latest"],
        ["148.0.7778.179", false, undefined],
      ]);
    });

    it("carries each version's own filename through to its row", () => {
      const rows = buildLibraryVersionRows({
        fleetMaintainedVersions: [
          v(1, "149.0.7827.54"),
          v(2, "148.0.7778.179"),
        ],
        activeVersion: "149.0.7827.54",
        pinnedVersion: null,
        addedTimestamp: "x",
      });
      expect(rows.map((r) => r.filename)).toEqual([
        "installer-149.0.7827.54.pkg",
        "installer-148.0.7778.179.pkg",
      ]);
    });

    it("badges the active row 'pinned' for an exact pin and 'majorVersion' for a caret pin", () => {
      const exact = buildLibraryVersionRows({
        fleetMaintainedVersions: [
          v(1, "149.0.7827.54"),
          v(2, "148.0.7778.179"),
        ],
        activeVersion: "148.0.7778.179",
        pinnedVersion: "148.0.7778.179",
        addedTimestamp: "x",
      });
      expect(exact.find((r) => r.isActive)?.badgeState).toBe("pinned");

      const major = buildLibraryVersionRows({
        fleetMaintainedVersions: [v(1, "149.0.7827.54")],
        activeVersion: "149.0.7827.54",
        pinnedVersion: "^149",
        addedTimestamp: "x",
      });
      expect(major[0].badgeState).toBe("majorVersion");
    });
  });

  describe("getPackageCardInfo", () => {
    it("returns the correct data for a software package (and without a custom display_name)", () => {
      const softwareTitle: ISoftwareTitleDetails = {
        id: 1,
        name: "Test Software",
        // display_name: undefined
        icon_url: "https://example.com/icon.png",
        versions: [{ id: 1, version: "1.0.0", vulnerabilities: [] }],
        software_package: {
          installer_id: 1,
          labels_include_any: null,
          labels_exclude_any: null,
          labels_include_all: null,
          name: "TestPackage.pkg",
          title_id: 2,
          version: "1.0.0",
          self_service: true,
          uploaded_at: "2021-01-01T00:00:00Z",
          status: {
            installed: 10,
            pending_install: 5,
            pending_uninstall: 3,
            failed_install: 2,
            failed_uninstall: 1,
          },
          install_script: "echo foo",
          uninstall_script: "echo bar",
          icon_url: "https://example.com/icon.png",
          automatic_install_policies: [],
          url: "",
        },
        packages: null,
        app_store_app: null,
        source: "apps",
        hosts_count: 10,
      };
      const packageCardInfo = getInstallerCardInfo(softwareTitle);
      expect(packageCardInfo).toEqual({
        softwareInstaller: softwareTitle.software_package,
        displayName: undefined,
        iconUrl: "https://example.com/icon.png",
        name: "TestPackage.pkg", // packages should display the package name not the software title name
        softwareDisplayName: "Test Software",
        version: "1.0.0",
        addedTimestamp: "2021-01-01T00:00:00Z",
        softwareTitleName: "Test Software",
        source: "apps",
        status: {
          installed: 10,
          pending: 8,
          failed: 3,
        },
        isScriptPackage: false,
        isSelfService: true,
      });
    });
    it("returns the correct data for an app store app (and with a custom display name)", () => {
      const softwareTitle: ISoftwareTitleDetails = {
        id: 1,
        name: "Test Software",
        display_name: "Test App",
        icon_url: "https://example.com/icon.png",
        versions: [{ id: 1, version: "1.0.0", vulnerabilities: [] }],
        software_package: null,
        packages: null,
        app_store_app: {
          app_store_id: "1",
          name: "Test App",
          display_name: "Test App",
          created_at: "2020-01-01T00:00:00.000Z",
          latest_version: "1.0.1",
          platform: "darwin",
          self_service: false,
          status: {
            installed: 10,
            pending: 5,
            failed: 3,
          },
          icon_url: "https://example.com/icon.png",
          labels_exclude_any: null,
          labels_include_any: null,
          labels_include_all: null,
        },
        source: "apps",
        hosts_count: 10,
      };
      const packageCardInfo = getInstallerCardInfo(softwareTitle);
      expect(packageCardInfo).toEqual({
        softwareInstaller: softwareTitle.app_store_app,
        name: "Test Software", // apps should display the software title name (backend should ensure the app name and software title name match)
        softwareDisplayName: "Test App",
        displayName: "Test App",
        iconUrl: "https://example.com/icon.png",
        version: "1.0.1",
        addedTimestamp: "2020-01-01T00:00:00.000Z",
        softwareTitleName: "Test Software",
        source: "apps",
        status: {
          installed: 10,
          pending: 5,
          failed: 3,
        },
        isScriptPackage: false,
        isSelfService: false,
      });
    });
  });
});
