import {
  FleetMaintainedAppPlatform,
  IFleetMaintainedApp,
} from "interfaces/software";

import fmaCatalog from "../../../../../../ee/maintained-apps/outputs/apps.json";

import {
  combineAppsByPlatform,
  WINDOWS_TO_MACOS_SLUGS,
} from "./FleetMaintainedAppsTable";

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

  it("shows the macOS name regardless of order when names differ otherwise", () => {
    const macos = app({
      id: 1,
      name: "Foo",
      slug: "foo/darwin",
      platform: "darwin",
    });
    const windows = app({
      id: 2,
      name: "Foo Desktop",
      slug: "foo/windows",
      platform: "windows",
    });

    [
      [macos, windows],
      [windows, macos],
    ].forEach((entries) => {
      const [combined] = combineAppsByPlatform(entries);
      expect(combined.displayName).toBe("Foo");
    });
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

  it("keeps two distinct apps that share a display name as separate rows", () => {
    // MacPaw Gemini and Google Gemini share the name "Gemini" but have
    // different slug tokens, so they must not collapse into one row.
    const combined = combineAppsByPlatform([
      app({ id: 1, name: "Gemini", slug: "gemini/darwin", platform: "darwin" }),
      app({
        id: 2,
        name: "Gemini",
        slug: "google-gemini/darwin",
        platform: "darwin",
      }),
    ]);

    expect(combined).toHaveLength(2);
    // Each row keeps its own macOS entry (neither Gemini is hidden/overwritten).
    expect(combined.map((c) => c.macos?.id).sort()).toEqual([1, 2]);
    expect(combined.map((c) => c.macos?.slug).sort()).toEqual([
      "gemini/darwin",
      "google-gemini/darwin",
    ]);
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

  it("reads apps.json and shows a macOS and a Windows Fleet-maintained app with the same name in one row", () => {
    // convert every app in apps.json to the shape the list API returns
    const catalogApps = fmaCatalog.apps.map((catalogApp, index) =>
      app({
        id: index,
        name: catalogApp.name,
        slug: catalogApp.slug,
        platform: catalogApp.platform as FleetMaintainedAppPlatform,
      })
    );

    // build the table rows the same way the UI does, and record each row's slugs as
    // "<macOS slug> + <Windows slug>", for example "docker-desktop/darwin + docker/windows"
    const combinedRowSlugs = new Set(
      combineAppsByPlatform(catalogApps).map(
        (row) => `${row.macos?.slug} + ${row.windows?.slug}`
      )
    );

    const errors: string[] = [];

    // group the apps by name, each name with one macOS and one Windows app should show as one row
    // R and WeChat use a different name on each platform, they share a slug token so they're
    // already combined and covered by the cases above
    const appsByName: Record<string, IFleetMaintainedApp[]> = {};
    catalogApps.forEach((catalogApp) => {
      appsByName[catalogApp.name] = appsByName[catalogApp.name] ?? [];
      appsByName[catalogApp.name].push(catalogApp);
    });

    Object.values(appsByName).forEach((appsWithSameName) => {
      // skip names that only one app uses
      if (appsWithSameName.length !== 2) {
        return;
      }

      // skip names that two apps on the same platform use
      const macosApp = appsWithSameName.find(
        (catalogApp) => catalogApp.platform === "darwin"
      );
      const windowsApp = appsWithSameName.find(
        (catalogApp) => catalogApp.platform === "windows"
      );
      if (!macosApp || !windowsApp) {
        return;
      }

      // look up the macOS and Windows app pair in the table rows, a missing pair means the UI
      // shows them as two separate rows
      if (!combinedRowSlugs.has(`${macosApp.slug} + ${windowsApp.slug}`)) {
        errors.push(
          `The Fleet-maintained apps ${macosApp.slug} and ${windowsApp.slug} use the same name ` +
            `'${macosApp.name}', but show as separate rows in the UI because their slugs are ` +
            `different. Add ['${windowsApp.slug}', '${macosApp.slug}'] to WINDOWS_TO_MACOS_SLUGS in ` +
            `FleetMaintainedAppsTable.tsx, or update this test if this message is incorrect.`
        );
      }
    });

    expect(errors).toEqual([]);
  });

  it("reads apps.json and finds both slugs of every pair in WINDOWS_TO_MACOS_SLUGS in it with the same name", () => {
    // look up each app's name by its slug, for example "docker/windows" -> "Docker Desktop"
    const nameBySlug = new Map(
      fmaCatalog.apps.map((catalogApp) => [catalogApp.slug, catalogApp.name])
    );

    const errors: string[] = [];

    // check every pair in WINDOWS_TO_MACOS_SLUGS, both slugs should be in apps.json with the
    // same name
    WINDOWS_TO_MACOS_SLUGS.forEach((macosSlug, windowsSlug) => {
      const windowsName = nameBySlug.get(windowsSlug);
      const macosName = nameBySlug.get(macosSlug);

      // a missing name means the slug isn't in apps.json
      if (windowsName === undefined || macosName === undefined) {
        errors.push(
          `WINDOWS_TO_MACOS_SLUGS in FleetMaintainedAppsTable.tsx combines ${windowsSlug} ` +
            `with ${macosSlug}, but apps.json doesn't have both. Remove or update this entry, or ` +
            `update this test if this message is incorrect.`
        );
        return;
      }

      if (windowsName !== macosName) {
        errors.push(
          `WINDOWS_TO_MACOS_SLUGS in FleetMaintainedAppsTable.tsx combines ${windowsSlug} ` +
            `('${windowsName}') with ${macosSlug} ('${macosName}'), but their names are ` +
            `different. Remove or update this entry, or update this test if this message is ` +
            `incorrect.`
        );
      }
    });

    expect(errors).toEqual([]);
  });
});
