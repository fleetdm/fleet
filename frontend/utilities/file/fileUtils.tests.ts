import { getPlatformDisplayName } from "./fileUtils";

describe("fileUtils", () => {
  describe("getPlatformDisplayName", () => {
    const testCases = [
      { extension: "pkg", platform: "macOS" },
      { extension: "json", platform: "macOS" },
      { extension: "mobileconfig", platform: "macOS" },
      { extension: "exe", platform: "Windows" },
      { extension: "msi", platform: "Windows" },
      { extension: "xml", platform: "Windows" },
      { extension: "deb", platform: "Linux" },
      { extension: "rpm", platform: "Linux" },
    ];

    testCases.forEach(({ extension, platform }) => {
      it(`should return ${platform} for .${extension} files`, () => {
        const file = new File([""], `test.${extension}`);
        expect(getPlatformDisplayName(file)).toEqual(platform);
      });
    });
  });
});
