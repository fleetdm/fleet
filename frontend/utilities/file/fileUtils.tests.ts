import { getPlatformDisplayName } from "./fileUtils";

describe("fileUtils", () => {
  describe("getPlatformDisplayName", () => {
    it("should return the correct platform display name depending on the file extension", () => {
      const file = new File([""], "test.pkg");
      expect(getPlatformDisplayName(file)).toEqual("macOS");

      const file2 = new File([""], "test.exe");
      expect(getPlatformDisplayName(file2)).toEqual("Windows");

      const file3 = new File([""], "test.deb");
      expect(getPlatformDisplayName(file3)).toEqual("linux");
    });
  });
});
