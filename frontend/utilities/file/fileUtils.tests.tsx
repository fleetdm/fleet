import {
  formatFileSize,
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
      { fileName: "test.py", expectedExtension: "py" },

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
        fileName: "test.py",
        expectedDetails: { name: "test.py", description: "macOS & Linux" },
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
      { extension: "py", platform: "macOS & Linux" },
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

  describe("fileUtils - formatFileSize", () => {
    // Expectations verified against the server's installersize.Human, which is
    // what writes the same limit into its own too-large error
    const testCases = [
      { bytes: 0, expectedSize: "0B" },
      { bytes: 999, expectedSize: "999B" },
      { bytes: 1000, expectedSize: "1kB" },
      { bytes: 1024, expectedSize: "1KiB" },
      { bytes: 1000000, expectedSize: "1MB" },
      { bytes: 1048576, expectedSize: "1MiB" },
      { bytes: 536870912, expectedSize: "512MiB" },
      { bytes: 1073741824, expectedSize: "1GiB" },
      { bytes: 1500000000, expectedSize: "1.5GB" },
      { bytes: 10737418240, expectedSize: "10GiB" },
      { bytes: 5497558138880, expectedSize: "5TiB" },
      { bytes: 1000000000000000, expectedSize: "1PB" },
      { bytes: 1125899906842624, expectedSize: "1PiB" },
    ];

    testCases.forEach(({ bytes, expectedSize }) => {
      it(`should return "${expectedSize}" for ${bytes} bytes`, () => {
        expect(formatFileSize(bytes)).toEqual(expectedSize);
      });
    });
  });
});
