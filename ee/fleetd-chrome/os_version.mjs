import Table from "./Table.js";

export default class TableOSVersion extends Table {
  name = "os_version";
  columns = ["platform", "platform_like", "version"];

  async generate(...args) {
    console.log("args", args);
    const data = await navigator.userAgentData.getHighEntropyValues([
      "architecture",
      "model",
      "platformVersion",
      "fullVersionList",
    ]);
    console.log(data);
    return [[data.platform, data.platform, data.platformVersion]];
  }
}
