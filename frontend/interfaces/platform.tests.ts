import {
  isAppleDevice,
  isAppleMdmOnlyPlatform,
  isDiskEncryptionSupportedLinuxPlatform,
  isIPadOrIPhone,
  isMacOS,
  isMobilePlatform,
  isNonQueryablePlatform,
  isWindows,
} from "./platform";

describe("platform helpers", () => {
  describe("tvOS", () => {
    it("is an Apple, MDM-only platform without osquery", () => {
      expect(isAppleDevice("tvos")).toBe(true);
      expect(isAppleMdmOnlyPlatform("tvos")).toBe(true);
      expect(isNonQueryablePlatform("tvos")).toBe(true);
    });

    it("is not treated as an iPhone, iPad, or mobile platform", () => {
      expect(isIPadOrIPhone("tvos")).toBe(false);
      expect(isMobilePlatform("tvos")).toBe(false);
    });
  });

  describe("isNonQueryablePlatform", () => {
    it.each(["ios", "ipados", "tvos", "android"])("matches %s", (platform) => {
      expect(isNonQueryablePlatform(platform)).toBe(true);
    });

    it.each(["darwin", "windows", "ubuntu", "chrome"])(
      "does not match %s",
      (platform) => {
        expect(isNonQueryablePlatform(platform)).toBe(false);
      }
    );
  });

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
    it("matches the distributions Fleet can escrow LUKS keys for", () => {
      expect(isDiskEncryptionSupportedLinuxPlatform("ubuntu", "")).toBe(true);
      expect(isDiskEncryptionSupportedLinuxPlatform("zorin", "")).toBe(true);
      expect(isDiskEncryptionSupportedLinuxPlatform("omarchy", "")).toBe(true);
    });

    it("matches Fedora only through its OS version, since it reports rhel", () => {
      expect(
        isDiskEncryptionSupportedLinuxPlatform("rhel", "Fedora Linux 41")
      ).toBe(true);
      expect(
        isDiskEncryptionSupportedLinuxPlatform("rhel", "CentOS Linux 7.9")
      ).toBe(false);
    });

    it("does not match other Linux distributions", () => {
      expect(isDiskEncryptionSupportedLinuxPlatform("arch", "")).toBe(false);
      expect(isDiskEncryptionSupportedLinuxPlatform("cachyos", "")).toBe(false);
      expect(isDiskEncryptionSupportedLinuxPlatform("manjaro", "")).toBe(false);
      expect(isDiskEncryptionSupportedLinuxPlatform("debian", "")).toBe(false);
      expect(isDiskEncryptionSupportedLinuxPlatform("pop", "")).toBe(false);
    });
  });
});
