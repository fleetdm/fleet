import Table from "./Table";

export default class TableOSVersion extends Table {
  name = "os_version";
  columns = [
    "name",
    "platform",
    "platform_like",
    "version",
    "major",
    "minor",
    "build",
    "patch",
    "arch",
    "codename",
  ];

  async generate() {
    // @ts-expect-error Typescript doesn't include the userAgentData API yet.
    const data = await navigator.userAgentData.getHighEntropyValues([
      "architecture",
      "model",
      "platformVersion",
      "fullVersionList",
    ]);

    let version = "";
    for (let entry of data.fullVersionList) {
      if (entry.brand === "Google Chrome") {
        version = entry.version;
        break;
      }
    }

    // Note MAJOR.MINOR.BUILD.PATCH (see https://www.chromium.org/developers/version-numbers/)
    const splits = version.split(".");
    let major = "",
      minor = "",
      build = "",
      patch = "";
    if (splits.length !== 4) {
      console.warn(
        `Chrome version ${version} does not have expected 4 segments`
      );
    } else {
      [major, minor, build, patch] = splits;
    }

    // Note we can actually get the platform of Chrome running on non-ChromeOS devices, but instead
    // we just hardcode to "chrome" so that Fleet always sees this Chrome extension as a Chrome
    // device even when we are doing local dev on a non-ChromeOS machine.
    const platform_info = await chrome.runtime.getPlatformInfo();
    const { arch } = platform_info;

    // Some of these values won't actually be correct on a non-chromeOS machine.
    return [
      {
        name: data.platform,
        platform: "chrome",
        platform_like: "chrome",
        version,
        major,
        minor,
        build,
        patch,
        codename: `Chrome OS ${data.platformVersion}`,
        arch: arch,
      },
    ];
  }
}
