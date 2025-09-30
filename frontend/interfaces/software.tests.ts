import { formatSoftwareType, ExtensionForType } from "./software";

describe("formatSoftwareType", () => {
  describe("basic source type conversion", () => {
    const testCases = [
      {
        source: "apps" as const,
        expected: "Application (macOS)",
        description: "macOS applications",
      },
      {
        source: "ios_apps" as const,
        expected: "Application (iOS)",
        description: "iOS applications",
      },
      {
        source: "ipados_apps" as const,
        expected: "Application (iPadOS)",
        description: "iPadOS applications",
      },
      {
        source: "programs" as const,
        expected: "Program (Windows)",
        description: "Windows programs",
      },
      {
        source: "deb_packages" as const,
        expected: "Package (deb)",
        description: "Debian packages",
      },
      {
        source: "rpm_packages" as const,
        expected: "Package (RPM)",
        description: "RPM packages",
      },
      {
        source: "npm_packages" as const,
        expected: "Package (NPM)",
        description: "NPM packages",
      },
      {
        source: "python_packages" as const,
        expected: "Package (Python)",
        description: "Python packages",
      },
      {
        source: "homebrew_packages" as const,
        expected: "Package (Homebrew)",
        description: "Homebrew packages",
      },
      {
        source: "chocolatey_packages" as const,
        expected: "Package (Chocolatey)",
        description: "Chocolatey packages",
      },
      {
        source: "pkg_packages" as const,
        expected: "Package (pkg)",
        description: "macOS pkg packages",
      },
    ];

    testCases.forEach(({ source, expected, description }) => {
      it(`should format ${description} correctly`, () => {
        expect(formatSoftwareType({ source })).toBe(expected);
      });
    });
  });

  describe("browser extensions with extension_for", () => {
    const testCases = [
      {
        source: "chrome_extensions" as const,
        extension_for: "chrome" as const,
        expected: "Browser plugin (Chrome)",
        description: "Chrome extensions",
      },
      {
        source: "chrome_extensions" as const,
        extension_for: "edge" as const,
        expected: "Browser plugin (Edge)",
        description: "Edge extensions",
      },
      {
        source: "chrome_extensions" as const,
        extension_for: "brave" as const,
        expected: "Browser plugin (Brave)",
        description: "Brave extensions",
      },
      {
        source: "chrome_extensions" as const,
        extension_for: "opera" as const,
        expected: "Browser plugin (Opera)",
        description: "Opera extensions",
      },
      {
        source: "chrome_extensions" as const,
        extension_for: "chromium" as const,
        expected: "Browser plugin (Chromium)",
        description: "Chromium extensions",
      },
      {
        source: "firefox_addons" as const,
        extension_for: undefined,
        expected: "Browser plugin (Firefox)",
        description: "Firefox add-ons without extension_for",
      },
      {
        source: "safari_extensions" as const,
        extension_for: undefined,
        expected: "Browser plugin (Safari)",
        description: "Safari extensions without extension_for",
      },
      {
        source: "ie_extensions" as const,
        extension_for: undefined,
        expected: "Browser plugin (IE)",
        description: "IE extensions without extension_for",
      },
    ];

    testCases.forEach(({ source, extension_for, expected, description }) => {
      it(`should format ${description} correctly`, () => {
        expect(formatSoftwareType({ source, extension_for })).toBe(expected);
      });
    });
  });

  describe("IDE extensions with extension_for", () => {
    const testCases = [
      {
        source: "vscode_extensions" as const,
        extension_for: "vscode" as const,
        expected: "IDE extension (VSCode)",
        description: "VSCode extensions",
      },
      {
        source: "vscode_extensions" as const,
        extension_for: "vscode_insiders" as const,
        expected: "IDE extension (VSCode Insiders)",
        description: "VSCode Insiders extensions",
      },
      {
        source: "vscode_extensions" as const,
        extension_for: "vscodium" as const,
        expected: "IDE extension (VSCodium)",
        description: "VSCodium extensions",
      },
      {
        source: "vscode_extensions" as const,
        extension_for: "cursor" as const,
        expected: "IDE extension (Cursor)",
        description: "Cursor extensions",
      },
      {
        source: "vscode_extensions" as const,
        extension_for: "trae" as const,
        expected: "IDE extension (Trae)",
        description: "Trae extensions",
      },
      {
        source: "vscode_extensions" as const,
        extension_for: "windsurf" as const,
        expected: "IDE extension (Windsurf)",
        description: "Windsurf extensions",
      },
    ];

    testCases.forEach(({ source, extension_for, expected, description }) => {
      it(`should format ${description} correctly`, () => {
        expect(formatSoftwareType({ source, extension_for })).toBe(expected);
      });
    });
  });

  describe("unknown extension_for values", () => {
    it("should use startCase for unknown extension_for values", () => {
      expect(
        formatSoftwareType({
          source: "chrome_extensions",
          extension_for: "unknown_browser" as ExtensionForType,
        })
      ).toBe("Browser plugin (Unknown Browser)");
    });

    it("should use startCase for unknown vscode extension_for values", () => {
      expect(
        formatSoftwareType({
          source: "vscode_extensions",
          extension_for: "unknown_editor" as ExtensionForType,
        })
      ).toBe("IDE extension (Unknown Editor)");
    });
  });

  describe("edge cases", () => {
    it("should handle unknown source types", () => {
      expect(
        formatSoftwareType({
          source: "unknown_source" as any,
        })
      ).toBe("Unknown");
    });

    it("should handle empty extension_for", () => {
      expect(
        formatSoftwareType({
          source: "chrome_extensions",
          extension_for: "",
        })
      ).toBe("Browser plugin");
    });

    it("should handle undefined extension_for", () => {
      expect(
        formatSoftwareType({
          source: "chrome_extensions",
          extension_for: undefined,
        })
      ).toBe("Browser plugin");
    });

    it("should handle null extension_for", () => {
      expect(
        formatSoftwareType({
          source: "chrome_extensions",
          extension_for: null as any,
        })
      ).toBe("Browser plugin");
    });
  });

  describe("all source types without extension_for", () => {
    const allSourceTypes = [
      "apt_sources",
      "deb_packages",
      "portage_packages",
      "rpm_packages",
      "yum_sources",
      "pacman_packages",
      "npm_packages",
      "atom_packages",
      "python_packages",
      "tgz_packages",
      "apps",
      "ios_apps",
      "ipados_apps",
      "chrome_extensions",
      "firefox_addons",
      "safari_extensions",
      "homebrew_packages",
      "programs",
      "ie_extensions",
      "chocolatey_packages",
      "pkg_packages",
      "vscode_extensions",
    ] as const;

    allSourceTypes.forEach((source) => {
      it(`should format ${source} without extension_for`, () => {
        const result = formatSoftwareType({ source });
        expect(result).toBeDefined();
        expect(typeof result).toBe("string");
        expect(result.length).toBeGreaterThan(0);
      });
    });
  });
});
