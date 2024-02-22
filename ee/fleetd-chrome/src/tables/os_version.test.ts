import VirtualDatabase from "../db";
import TableOSVersion from "./os_version";

describe("os_version", () => {
  describe("getName", () => {
    const sut = new TableOSVersion(null, null);
    it("returns platform name properly formatted", () => {
      expect(sut.getName("Chrome OS")).toBe("ChromeOS");
    });
  });

  describe("getCodename", () => {
    const sut = new TableOSVersion(null, null);
    it("has the proper prefix", () => {
      expect(sut.getCodename("10.0.0").startsWith("ChromeOS")).toBe(true);
    });
  });

  test("success", async () => {
    // @ts-expect-error Typescript doesn't include the userAgentData API yet.
    global.navigator.userAgentData = {
      getHighEntropyValues: jest.fn(() =>
        Promise.resolve({
          architecture: "x86",
          fullVersionList: [
            { brand: "Chromium", version: "110.0.5481.177" },
            { brand: "Not A(Brand", version: "24.0.0.0" },
            { brand: "Google Chrome", version: "110.0.5481.177" },
          ],
          mobile: false,
          model: "",
          platform: "Chrome OS",
          platformVersion: "13.2.1",
        })
      ),
    };
    chrome.runtime.getPlatformInfo = jest.fn(() =>
      Promise.resolve({ os: "cros", arch: "x86-64", nacl_arch: "x86-64" })
    );

    const db = await VirtualDatabase.init();
    globalThis.DB = db;

    const res = await db.query("select * from os_version");
    expect(res).toEqual({
      data: [
        {
          name: "ChromeOS",
          platform: "chrome",
          platform_like: "chrome",
          version: "110.0.5481.177",
          major: "110",
          minor: "0",
          build: "5481",
          patch: "177",
          arch: "x86-64",
          codename: "ChromeOS 13.2.1",
        },
      ],
      warnings: "",
    });
  });

  test("unexpected version string", async () => {
    // @ts-expect-error Typescript doesn't include the userAgentData API yet.
    global.navigator.userAgentData = {
      getHighEntropyValues: jest.fn(() =>
        Promise.resolve({
          architecture: "x86",
          fullVersionList: [
            { brand: "Chromium", version: "110.0.5481.177" },
            { brand: "Not A(Brand", version: "24.0.0.0" },
            { brand: "Google Chrome", version: "110.weird_version" },
          ],
          mobile: false,
          model: "",
          platform: "Chrome OS",
          platformVersion: "13.2.1",
        })
      ),
    };
    chrome.runtime.getPlatformInfo = jest.fn(() =>
      Promise.resolve({ os: "cros", arch: "x86-64", nacl_arch: "x86-64" })
    );
    console.warn = jest.fn();

    const db = await VirtualDatabase.init();
    globalThis.DB = db;
    const res = await db.query("select * from os_version");
    expect(res).toEqual({
      data: [
        {
          name: "ChromeOS",
          platform: "chrome",
          platform_like: "chrome",
          version: "110.weird_version",
          major: "",
          minor: "",
          build: "",
          patch: "",
          arch: "x86-64",
          codename: "ChromeOS 13.2.1",
        },
      ],
      warnings: "",
    });
    expect(console.warn).toHaveBeenCalledWith(
      expect.stringContaining("expected 4 segments")
    );
  });

  test("not even chrome", async () => {
    // @ts-expect-error Typescript doesn't include the userAgentData API yet.
    global.navigator.userAgentData = {
      getHighEntropyValues: jest.fn(() =>
        Promise.resolve({
          fullVersionList: [
            { brand: "Not even Chrome", version: "103.0.5060.134" },
            { brand: "Not chrome", version: "103.0.5060.134" },
          ],
        })
      ),
    };

    const db = await VirtualDatabase.init();
    globalThis.DB = db;

    const res = await db.query("select * from os_version");
    expect(res.warnings).toContain("environment does not look like Chrome");
  });
});
