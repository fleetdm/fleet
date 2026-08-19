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
});
