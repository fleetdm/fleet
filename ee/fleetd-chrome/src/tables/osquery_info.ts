import Table from "./Table";

export default class TableOsqueryInfo extends Table {
  name = "osquery_info";
  columns = ["version", "build_platform", "build_distro", "extensions"];

  async generate() {
    return {
      data: [
        {
          version: `fleetd-chrome-${chrome.runtime.getManifest().version}`,
          build_platform: "chrome",
          build_distro: "chrome",
          extensions: "inactive",
        },
      ],
    };
  }
}
