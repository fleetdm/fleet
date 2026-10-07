import {
  formatSoftwareType,
  formatSoftwareVersion,
  getSoftwareTypesForPlatform,
  parseSoftwareTypesParam,
  SOFTWARE_TYPES,
  SoftwareExtensionFor,
  SoftwareSource,
  softwareTypesToApiParams,
} from "./software";

describe("formatSoftwareType", () => {
  it("matches JetBrains products by the IDE name osquery reports", () => {
    expect(
      formatSoftwareType({
        source: "jetbrains_plugins",
        extension_for: "IntelliJIdea",
      })
    ).toBe("IntelliJ IDEA extension");
  });

  it("falls back to a generic label when extension_for is empty", () => {
    expect(formatSoftwareType({ source: "chrome_extensions" })).toBe(
      "Browser extension"
    );
    expect(
      formatSoftwareType({ source: "firefox_addons", extension_for: "" })
    ).toBe("Browser extension");
    expect(formatSoftwareType({ source: "vscode_extensions" })).toBe(
      "IDE extension"
    );
    expect(formatSoftwareType({ source: "jetbrains_plugins" })).toBe(
      "IDE extension"
    );
  });

  it("ignores extension_for on sources that aren't extensions", () => {
    expect(
      formatSoftwareType({
        source: "apps",
        extension_for: "chrome",
      })
    ).toBe("macOS app");
  });

  it("names unknown browsers and IDEs after their extension_for value", () => {
    expect(
      formatSoftwareType({
        source: "chrome_extensions",
        extension_for: "arc" as SoftwareExtensionFor,
      })
    ).toBe("arc extension");
  });

  it("keeps labels for legacy sources", () => {
    expect(formatSoftwareType({ source: "apt_sources" })).toBe("APT package");
  });

  it("returns Unknown for unknown sources", () => {
    expect(
      formatSoftwareType({ source: "unknown_source" as SoftwareSource })
    ).toBe("Unknown");
  });

  it("labels every catalog type with its display name", () => {
    SOFTWARE_TYPES.forEach((t) => {
      expect(
        formatSoftwareType({ source: t.source, extension_for: t.extensionFor })
      ).toBe(t.displayName);
    });
  });
});

describe("getSoftwareTypesForPlatform", () => {
  const keysFor = (platform: string) =>
    getSoftwareTypesForPlatform(platform, { hostPage: true }).map((t) => t.key);

  it.each(["darwin", "windows", "ubuntu", "chrome", "ios", "android"])(
    "hides installer-only types on %s host pages",
    (platform) => {
      expect(
        getSoftwareTypesForPlatform(platform, { hostPage: true }).some(
          (t) => t.installerOnly
        )
      ).toBe(false);
    }
  );

  // A case-sensitive sort would push "deb package", "macOS app" and
  // "npm package" below every capitalized name.
  it("sorts types case-insensitively by display name", () => {
    const names = (platform: string) =>
      getSoftwareTypesForPlatform(platform).map((t) => t.displayName);
    expect(names("ubuntu").indexOf("deb package")).toBeLessThan(
      names("ubuntu").indexOf("Edge extension")
    );
    expect(names("darwin").indexOf("macOS app")).toBeLessThan(
      names("darwin").indexOf("Opera extension")
    );
    expect(names("darwin").indexOf("npm package")).toBeLessThan(
      names("darwin").indexOf("Opera extension")
    );
  });

  it("includes installer-only types outside host pages", () => {
    expect(
      getSoftwareTypesForPlatform("darwin").some((t) => t.installerOnly)
    ).toBe(true);
  });

  it("lists every Linux package type on every Linux host", () => {
    const linuxPackageKeys = [
      "deb_package",
      "rpm_package",
      "pacman_package",
      "portage_package",
    ];
    ["ubuntu", "rhel", "arch", "linux"].forEach((platform) => {
      expect(keysFor(platform)).toEqual(
        expect.arrayContaining(linuxPackageKeys)
      );
    });
    expect(keysFor("darwin")).not.toContain("deb_package");
  });

  it("returns no types for an unknown platform", () => {
    expect(keysFor("unknown")).toEqual([]);
  });
});

describe("softwareTypesToApiParams", () => {
  it("translates type keys into source and extension_for", () => {
    expect(
      softwareTypesToApiParams([
        "macos_app",
        "brave_extension",
        "cursor_extension",
      ])
    ).toEqual({
      source: "apps,chrome_extensions,vscode_extensions",
      extension_for: "brave,cursor",
    });
  });

  it("omits extension_for when no extension type is selected", () => {
    expect(softwareTypesToApiParams(["macos_app", "deb_package"])).toEqual({
      source: "apps,deb_packages",
    });
  });

  it("dedupes sources and ignores unknown keys", () => {
    expect(
      softwareTypesToApiParams(["chrome_extension", "edge_extension", "foo"])
    ).toEqual({ source: "chrome_extensions", extension_for: "chrome,edge" });
  });

  it("returns no params for an empty selection", () => {
    expect(softwareTypesToApiParams([])).toEqual({});
  });
});

describe("parseSoftwareTypesParam", () => {
  it("returns undefined when the param is absent", () => {
    expect(parseSoftwareTypesParam(undefined)).toBeUndefined();
  });

  it("drops unknown keys", () => {
    expect(parseSoftwareTypesParam("foo,macos_app")).toEqual(["macos_app"]);
    expect(parseSoftwareTypesParam("foo")).toEqual([]);
  });

  it("returns no types for an empty param", () => {
    expect(parseSoftwareTypesParam("")).toEqual([]);
  });

  it("ignores whitespace and empty entries", () => {
    expect(parseSoftwareTypesParam(" macos_app , ,brave_extension,")).toEqual([
      "macos_app",
      "brave_extension",
    ]);
  });

  it("accepts a repeated param", () => {
    expect(
      parseSoftwareTypesParam(["macos_app", "foo,brave_extension"])
    ).toEqual(["macos_app", "brave_extension"]);
  });

  it("dedupes keys", () => {
    expect(parseSoftwareTypesParam("macos_app,macos_app")).toEqual([
      "macos_app",
    ]);
  });
});

describe("formatSoftwareVersion", () => {
  const testCases = [
    {
      version: "v0.21.1",
      release: "go1.26.1",
      source: "go_binaries",
      expected: "v0.21.1 (go1.26.1)",
      description: "a Go binary with a toolchain version",
    },
    {
      version: "v0.21.1",
      release: "",
      source: "go_binaries",
      expected: "v0.21.1",
      description: "a Go binary with an empty toolchain version",
    },
    {
      version: "v0.21.1",
      release: undefined,
      source: "go_binaries",
      expected: "v0.21.1",
      description: "a Go binary with no toolchain version",
    },
    {
      version: "(devel)",
      release: "go1.26.1",
      source: "go_binaries",
      expected: "(devel) (go1.26.1)",
      description: "a binary built with `go build`",
    },
    {
      version: "1.2.3",
      release: "30.el7",
      source: "rpm_packages",
      expected: "1.2.3",
      description: "an RPM package with a package release",
    },
    {
      version: "1.2.3",
      release: undefined,
      source: undefined,
      expected: "1.2.3",
      description: "a version with no source",
    },
  ];

  testCases.forEach(({ version, release, source, expected, description }) => {
    it(`should format ${description} correctly`, () => {
      expect(formatSoftwareVersion({ version, release, source })).toBe(
        expected
      );
    });
  });
});
