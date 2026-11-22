import { IFleetMaintainedApp } from "interfaces/software";

import { combineAppsByPlatform } from "./FleetMaintainedAppsTable";

const app = (overrides: Partial<IFleetMaintainedApp>): IFleetMaintainedApp => ({
  id: 1,
  name: "App",
  version: "1.0",
  platform: "darwin",
  slug: "app/darwin",
  ...overrides,
});

describe("combineAppsByPlatform", () => {
  it("combines an app's macOS and Windows entries into a single row", () => {
    const combined = combineAppsByPlatform([
      app({ id: 1, name: "Figma", slug: "figma/darwin", platform: "darwin" }),
      app({ id: 2, name: "Figma", slug: "figma/windows", platform: "windows" }),
    ]);

    expect(combined).toHaveLength(1);
    expect(combined[0].name).toBe("Figma");
    expect(combined[0].macos?.id).toBe(1);
    expect(combined[0].windows?.id).toBe(2);
  });

  it("shows the base name regardless of the order of the entries", () => {
    const macos = app({
      id: 1,
      name: "R for macOS",
      slug: "r/darwin",
      platform: "darwin",
    });
    const windows = app({
      id: 2,
      name: "R for Windows",
      slug: "r/windows",
      platform: "windows",
    });

    [
      [macos, windows],
      [windows, macos],
    ].forEach((entries) => {
      const [combined] = combineAppsByPlatform(entries);
      expect(combined.name).toBe("R for macOS");
      expect(combined.displayName).toBe("R");
    });
  });

  it("shows the base name when only one platform's name has a suffix", () => {
    const [combined] = combineAppsByPlatform([
      app({
        id: 1,
        name: "WeChat",
        slug: "wechat/windows",
        platform: "windows",
      }),
      app({
        id: 2,
        name: "WeChat for Mac",
        slug: "wechat/darwin",
        platform: "darwin",
      }),
    ]);

    expect(combined.name).toBe("WeChat for Mac");
    expect(combined.displayName).toBe("WeChat");
  });

  it("keeps the suffix in a single-platform app's name", () => {
    const [combined] = combineAppsByPlatform([
      app({
        id: 1,
        name: "Google Credential Provider for Windows",
        slug: "google-credential-provider/windows",
        platform: "windows",
      }),
    ]);

    expect(combined.displayName).toBe("Google Credential Provider for Windows");
  });

  it("combines macOS and Windows entries whose names differ only by case into one row", () => {
    const combined = combineAppsByPlatform([
      app({
        id: 1,
        name: "Github Desktop",
        slug: "github/darwin",
        platform: "darwin",
      }),
      app({
        id: 2,
        name: "GitHub Desktop",
        slug: "github-desktop/windows",
        platform: "windows",
      }),
    ]);

    expect(combined).toHaveLength(1);
    expect(combined[0].displayName).toBe("Github Desktop");
    expect(combined[0].windows?.id).toBe(2);
  });

  it("shows Foo and Foo Desktop as separate rows when they share a slug token but not a name", () => {
    const combined = combineAppsByPlatform([
      app({
        id: 1,
        name: "Foo",
        slug: "foo/darwin",
        platform: "darwin",
      }),
      app({
        id: 2,
        name: "Foo Desktop",
        slug: "foo/windows",
        platform: "windows",
      }),
    ]);

    expect(combined).toHaveLength(2);
    expect(combined[0].name).toBe("Foo");
    expect(combined[1].name).toBe("Foo Desktop");
  });

  it("combines Docker Desktop's macOS and Windows entries into one row when their slug tokens differ", () => {
    const macos = app({
      id: 1,
      name: "Docker Desktop",
      slug: "docker-desktop/darwin",
      platform: "darwin",
    });
    const windows = app({
      id: 2,
      name: "Docker Desktop",
      slug: "docker/windows",
      platform: "windows",
    });

    // pass the entries in both orders, each order should give one row that keeps both slugs
    [
      [macos, windows],
      [windows, macos],
    ].forEach((entries) => {
      const combined = combineAppsByPlatform(entries);
      expect(combined).toHaveLength(1);
      expect(combined[0].macos?.slug).toBe("docker-desktop/darwin");
      expect(combined[0].windows?.slug).toBe("docker/windows");
    });
  });
});
