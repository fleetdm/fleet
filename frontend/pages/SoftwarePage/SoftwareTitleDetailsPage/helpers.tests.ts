import { ISoftwareTitleDetails } from "interfaces/software";
import { getInstallerCardInfo } from "./helpers";

describe("SoftwareTitleDetailsPage helpers", () => {
  describe("getPackageCardInfo", () => {
    it("returns the correct data for a software package", () => {
      const softwareTitle: ISoftwareTitleDetails = {
        id: 1,
        name: "Test Software",
        versions: [{ id: 1, version: "1.0.0", vulnerabilities: [] }],
        software_package: {
          labels_include_any: null,
          labels_exclude_any: null,
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
        app_store_app: null,
        source: "apps",
        hosts_count: 10,
      };
      const packageCardInfo = getInstallerCardInfo(softwareTitle);
      expect(packageCardInfo).toEqual({
        softwarePackage: softwareTitle.software_package,
        name: "TestPackage.pkg", // packages should display the package name not the software title name
        version: "1.0.0",
        addedTimestamp: "2021-01-01T00:00:00Z",
        softwareTitleName: "Test Software",
        status: {
          installed: 10,
          pending: 8,
          failed: 3,
        },
        isSelfService: true,
      });
    });
    it("returns the correct data for an app store app", () => {
      const softwareTitle: ISoftwareTitleDetails = {
        id: 1,
        name: "Test Software",
        versions: [{ id: 1, version: "1.0.0", vulnerabilities: [] }],
        software_package: null,
        app_store_app: {
          app_store_id: 1,
          name: "Test App",
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
        },
        source: "apps",
        hosts_count: 10,
      };
      const packageCardInfo = getInstallerCardInfo(softwareTitle);
      expect(packageCardInfo).toEqual({
        softwarePackage: softwareTitle.app_store_app,
        name: "Test Software", // apps should display the software title name (backend should ensure the app name and software title name match)
        version: "1.0.1",
        addedTimestamp: "2020-01-01T00:00:00.000Z",
        softwareTitleName: "Test Software",
        status: {
          installed: 10,
          pending: 5,
          failed: 3,
        },
        isSelfService: false,
      });
    });
  });
});
