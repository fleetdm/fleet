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

  getName(platform: string): string {
    return platform.replace("Chrome OS", "ChromeOS");
  }

  getCodename(platformVersion: string): string {
    return `ChromeOS ${platformVersion}`;
  }

  async generate() {
    // @ts-expect-error Typescript doesn't include the userAgentData API yet.
    const data = await navigator.userAgentData.getHighEntropyValues([
      "fullVersionList",
      "platform",
      "platformVersion",
    ]);

    let version = "";
    for (let entry of data.fullVersionList) {
      if (entry.brand === "Google Chrome") {
        version = entry.version;
        break;
      }
    }
    if (version === "") {
      throw new Error("environment does not look like Chrome");
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
    const platformInfo = await chrome.runtime.getPlatformInfo();
    const { arch } = platformInfo;

    // Some of these values won't actually be correct on a non-chromeOS machine.
    return {
      data: [
        {
          name: this.getName(data.platform),
          platform: "chrome",
          platform_like: "chrome",
          version,
          major,
          minor,
          build,
          patch,
          codename: this.getCodename(data.platformVersion),
          // https://developer.chrome.com/docs/extensions/reference/runtime/#type-PlatformArch
          arch: arch,
        },
      ],
    };
  }
}
