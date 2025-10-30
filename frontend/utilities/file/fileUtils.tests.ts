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
        expectedDetails: { name: "test.pkg", description: "macOS" },
      },
      {
        fileName: "test.exe",
        expectedDetails: { name: "test.exe", description: "Windows" },
      },
      {
        fileName: "test.tar.gz",
        expectedDetails: { name: "test.tar.gz", description: "Linux" },
      },
      {
        fileName: "unknown.file",
        expectedDetails: { name: "unknown.file", description: undefined },
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
    expect(getFileDetails(file)).toEqual({ name: "", description: undefined });
  });

  it("should handle files with no extension gracefully", () => {
    const file = new File([""], `no_extension`);
    expect(getPlatformDisplayName(file)).toBeUndefined();
    expect(getFileDetails(file)).toEqual({
      name: "no_extension",
      description: undefined,
    });
  });

  it("should handle filenames with multiple dots correctly", () => {
    const file = new File([""], `my.file.name.pkg`);
    expect(getPlatformDisplayName(file)).toEqual("macOS");
    expect(getFileDetails(file)).toEqual({
      name: "my.file.name.pkg",
      description: "macOS",
    });
  });

  describe("fileUtils - isSoftwareInstaller parameter", () => {
    it('should return "macOS & Linux" for .sh files when isSoftwareInstaller is false (default)', () => {
      const file = new File([""], "script.sh");
      expect(getPlatformDisplayName(file)).toEqual("macOS & Linux");
      expect(getPlatformDisplayName(file, false)).toEqual("macOS & Linux");
      expect(getFileDetails(file)).toEqual({
        name: "script.sh",
        description: "macOS & Linux",
      });
      expect(getFileDetails(file, false)).toEqual({
        name: "script.sh",
        description: "macOS & Linux",
      });
    });

    it('should return "Linux" for .sh files when isSoftwareInstaller is true', () => {
      const file = new File([""], "installer.sh");
      expect(getPlatformDisplayName(file, true)).toEqual("Linux");
      expect(getFileDetails(file, true)).toEqual({
        name: "installer.sh",
        description: "Linux",
      });
    });

    it("should not affect other file extensions when isSoftwareInstaller is true", () => {
      const testCases = [
        { fileName: "test.pkg", expected: "macOS" },
        { fileName: "test.exe", expected: "Windows" },
        { fileName: "test.ps1", expected: "Windows" },
        { fileName: "test.deb", expected: "Linux" },
      ];

      testCases.forEach(({ fileName, expected }) => {
        const file = new File([""], fileName);
        // Should return same value regardless of isSoftwareInstaller
        expect(getPlatformDisplayName(file, false)).toEqual(expected);
        expect(getPlatformDisplayName(file, true)).toEqual(expected);
      });
    });
  });
});
