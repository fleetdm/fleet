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
  it.each<[SoftwareSource, SoftwareExtensionFor | undefined, string]>([
    ["apps", undefined, "macOS app"],
    ["deb_packages", "", "deb package"],
    ["chrome_extensions", "brave", "Brave extension"],
    ["vscode_extensions", "vscode", "VS Code extension"],
    ["jetbrains_plugins", "IntelliJIdea", "IntelliJ IDEA extension"],
  ])("formats %s / %s as %s", (source, extension_for, expected) => {
    expect(formatSoftwareType({ source, extension_for })).toBe(expected);
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
    expect(formatSoftwareType({ source: "yum_sources" })).toBe("YUM package");
    expect(formatSoftwareType({ source: "atom_packages" })).toBe(
      "Atom package"
    );
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

describe("SOFTWARE_TYPES", () => {
  it("has 50 types with unique keys", () => {
    expect(SOFTWARE_TYPES).toHaveLength(50);
    expect(new Set(SOFTWARE_TYPES.map((t) => t.key)).size).toBe(50);
  });

  it("sorts case-insensitively by display name", () => {
    const names = SOFTWARE_TYPES.map((t) => t.displayName);
    const lowerNames = names.map((name) => name.toLowerCase());
    expect(lowerNames).toEqual([...lowerNames].sort());
  });
});

describe("getSoftwareTypesForPlatform", () => {
  const installerOnlyKeys = [
    "macos_package_pkg",
    "tarball_tar_gz",
    "script_only_package_py",
    "script_only_package_sh",
    "script_only_package_ps1",
  ];
  const keysFor = (platform: string) =>
    getSoftwareTypesForPlatform(platform, { hostPage: true }).map((t) => t.key);

  it.each([
    ["darwin", 35],
    ["windows", 34],
    ["ubuntu", 33],
    ["rhel", 32],
    ["arch", 33],
    ["gentoo", 33],
    ["linux", 33],
    ["chrome", 1],
    ["ios", 1],
    ["ipados", 1],
    ["android", 1],
  ])("lists %s host types", (platform, count) => {
    const keys = keysFor(platform);
    expect(keys).toHaveLength(count);
    installerOnlyKeys.forEach((key) => expect(keys).not.toContain(key));
  });

  it("includes installer-only types outside host pages", () => {
    const keys = getSoftwareTypesForPlatform("darwin").map((t) => t.key);
    expect(keys).toEqual(
      expect.arrayContaining([
        "macos_package_pkg",
        "script_only_package_py",
        "script_only_package_sh",
      ])
    );
  });

  // Per the planning sheet, RPM package is listed on deb hosts but deb package
  // is not listed on rpm hosts.
  it("lists the platform's package managers", () => {
    expect(keysFor("chrome")).toEqual(["chrome_extension"]);
    expect(keysFor("ubuntu")).toEqual(
      expect.arrayContaining(["deb_package", "rpm_package"])
    );
    expect(keysFor("rhel")).toContain("rpm_package");
    expect(keysFor("rhel")).not.toContain("deb_package");
    expect(keysFor("arch")).toEqual(
      expect.arrayContaining(["pacman_package", "portage_package"])
    );
    expect(keysFor("arch")).not.toContain("rpm_package");
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
