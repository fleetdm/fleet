import {
  getExtensionFromFileName,
  getFileDetails,
  getPlatformDisplayName,
} from "./fileUtils";

describe("fileUtils", () => {
  describe("fileUtils - getExtensionFromFileName", () => {
    const testCases = [
      // Simple extensions
      { fileName: "test.pkg", expectedExtension: "pkg" },
      { fileName: "test.json", expectedExtension: "json" },
      { fileName: "test.mobileconfig", expectedExtension: "mobileconfig" },
      { fileName: "test.exe", expectedExtension: "exe" },
      { fileName: "test.msi", expectedExtension: "msi" },
      { fileName: "test.xml", expectedExtension: "xml" },
      { fileName: "test.deb", expectedExtension: "deb" },
      { fileName: "test.rpm", expectedExtension: "rpm" },
      { fileName: "test.tar", expectedExtension: "tar" },

      // Compound extensions
      { fileName: "test.tar.gz", expectedExtension: "tar.gz" },
      { fileName: "test.tar.xz", expectedExtension: "tar.xz" },
      { fileName: "test.tar.bz2", expectedExtension: "tar.bz2" },
      { fileName: "test.tar.zst", expectedExtension: "tar.zst" },

      // Alias for compound extensions
      { fileName: "test.tgz", expectedExtension: "tar.gz" },
      { fileName: "test.tbz2", expectedExtension: "tar.bz2" },
      { fileName: "test.tzst", expectedExtension: "tar.zst" },
      { fileName: "test.txz", expectedExtension: "tar.xz" },

      // No extension
      { fileName: "no_extension", expectedExtension: undefined },
    ];

    testCases.forEach(({ fileName, expectedExtension }) => {
      it(`should return "${expectedExtension}" for "${fileName}"`, () => {
        expect(getExtensionFromFileName(fileName)).toEqual(expectedExtension);
      });
    });
  });

  describe("fileUtils - getFileDetails", () => {
    const testCases = [
      {
        fileName: "test.pkg",
        expectedDetails: { name: "test.pkg", platform: "macOS" },
      },
      {
        fileName: "test.exe",
        expectedDetails: { name: "test.exe", platform: "Windows" },
      },
      {
        fileName: "test.tar.gz",
        expectedDetails: { name: "test.tar.gz", platform: "Linux" },
      },
      {
        fileName: "unknown.file",
        expectedDetails: { name: "unknown.file", platform: undefined },
      },
    ];

    testCases.forEach(({ fileName, expectedDetails }) => {
      it(`should return correct details for "${fileName}"`, () => {
        const file = new File([""], fileName);
        expect(getFileDetails(file)).toEqual(expectedDetails);
      });
    });
  });

  describe("fileUtils - getPlatformDisplayName", () => {
    const testCases = [
      { extension: "pkg", platform: "macOS" },
      { extension: "json", platform: "macOS" },
      { extension: "mobileconfig", platform: "macOS" },
      { extension: "exe", platform: "Windows" },
      { extension: "msi", platform: "Windows" },
      { extension: "xml", platform: "Windows" },
      { extension: "deb", platform: "Linux" },
      { extension: "tar.gz", platform: "Linux" },
      { extension: undefined, platform: undefined }, // no extension
      { extension: "unknown_ext", platform: undefined }, // unmapped extension
    ];

    testCases.forEach(({ extension, platform }) => {
      it(`should return "${platform}" for ".${extension}" files`, () => {
        const file = new File([""], `test.${extension}`);
        expect(getPlatformDisplayName(file)).toEqual(platform);
      });
    });
  });

  it("should handle empty filenames gracefully", () => {
    const file = new File([""], "");
    expect(getPlatformDisplayName(file)).toBeUndefined();
    expect(getFileDetails(file)).toEqual({ name: "", platform: undefined });
  });

  it("should handle files with no extension gracefully", () => {
    const file = new File([""], `no_extension`);
    expect(getPlatformDisplayName(file)).toBeUndefined();
    expect(getFileDetails(file)).toEqual({
      name: "no_extension",
      platform: undefined,
    });
  });

  it("should handle filenames with multiple dots correctly", () => {
    const file = new File([""], `my.file.name.pkg`);
    expect(getPlatformDisplayName(file)).toEqual("macOS");
    expect(getFileDetails(file)).toEqual({
      name: "my.file.name.pkg",
      platform: "macOS",
    });
  });
});
