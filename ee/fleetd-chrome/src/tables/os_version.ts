import Table from "./Table";

export default class TableOSVersion extends Table {
  name = "os_version";
  columns = ["name", "platform", "platform_like", "version", "build", "arch"];

  async generate() {
    // @ts-expect-error Typescript doesn't include the userAgentData API yet.
    const data = await navigator.userAgentData.getHighEntropyValues([
      "architecture",
      "model",
      "platformVersion",
      "fullVersionList",
    ]);

    // Note we can actually get the platform of Chrome running on non-ChromeOS devices, but instead
    // we just hardcode to "chrome" so that Fleet always sees this Chrome extension as a Chrome
    // device even when we are doing local dev on a non-ChromeOS machine.
    const platform_info = await chrome.runtime.getPlatformInfo();
    const { arch } = platform_info;

    return [
      {
        name: data.platform,
        platform: "chrome",
        platform_like: "chrome",
        version: data.platformVersion,
        build: data.platformVersion,
        arch: arch,
      },
    ];
  }
}
