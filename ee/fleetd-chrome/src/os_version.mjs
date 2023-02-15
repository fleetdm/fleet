import Table from "./Table.js";

export default class TableOSVersion extends Table {
  name = "os_version";
  columns = ["name", "platform", "platform_like", "version", "build", "arch"];

  async generate(...args) {
    const data = await navigator.userAgentData.getHighEntropyValues([
      "architecture",
      "model",
      "platformVersion",
      "fullVersionList",
    ]);

    const platform_info = await chrome.runtime.getPlatformInfo();
    const { arch, os: platform } = platform_info;

    return [
      [
        data.platform,
        platform,
        platform,
        data.platformVersion,
        data.platformVersion,
        arch,
      ],
    ];
  }
}
