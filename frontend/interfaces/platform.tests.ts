import { isMacOS, isWindows } from "./platform";

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
});
