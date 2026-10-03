import {
  HostPlatform,
  isDiskEncryptionSupportedLinuxPlatform,
  isMacOS,
  isWindows,
} from "./platform";

describe("platform helpers", () => {
  describe("isMacOS", () => {
    it("matches a single darwin platform string", () => {
      expect(isMacOS("darwin")).toBe(true);
    });

    it("matches the legacy macos platform string", () => {
      expect(isMacOS("macos")).toBe(true);
    });

    it("matches a comma-joined platform string containing darwin", () => {
      expect(isMacOS("darwin,linux")).toBe(true);
      expect(isMacOS("windows,darwin")).toBe(true);
    });

    it("does not match Windows or Linux singletons", () => {
      expect(isMacOS("windows")).toBe(false);
      expect(isMacOS("linux")).toBe(false);
    });

    it("does not match an empty string", () => {
      expect(isMacOS("")).toBe(false);
    });
  });

  describe("isWindows", () => {
    it("matches a single windows platform string", () => {
      expect(isWindows("windows")).toBe(true);
    });

    it("matches a comma-joined platform string containing windows", () => {
      expect(isWindows("windows,linux")).toBe(true);
      expect(isWindows("darwin,windows")).toBe(true);
    });

    it("does not match Darwin or Linux singletons", () => {
      expect(isWindows("darwin")).toBe(false);
      expect(isWindows("linux")).toBe(false);
    });

    it("does not match an empty string", () => {
      expect(isWindows("")).toBe(false);
    });
  });

  describe("isDiskEncryptionSupportedLinuxPlatform", () => {
    it.each<[HostPlatform, string]>([
      ["ubuntu", "Ubuntu 24.04.1 LTS"],
      ["zorin", "Zorin OS 17.2"],
      ["rhel", "Fedora Linux 41.0.0"],
      ["arch", "Arch Linux rolling"],
      ["archarm", "Arch Linux ARM rolling"],
      ["manjaro", "Manjaro Linux 25.0.0"],
      ["manjaro-arm", "Manjaro ARM 25.0.0"],
      ["cachyos", "CachyOS Linux rolling"],
      ["omarchy", "Omarchy 4.0.0"],
    ])("matches %s (%s)", (platform, osVersion) => {
      expect(isDiskEncryptionSupportedLinuxPlatform(platform, osVersion)).toBe(
        true
      );
    });

    it.each<[HostPlatform, string]>([
      ["rhel", "CentOS Linux 7.9.2009"],
      ["debian", "Debian GNU/Linux 12"],
      ["pop", "Pop!_OS 22.04 LTS"],
      ["darwin", "macOS 15.1"],
    ])("does not match %s (%s)", (platform, osVersion) => {
      expect(isDiskEncryptionSupportedLinuxPlatform(platform, osVersion)).toBe(
        false
      );
    });
  });
});
