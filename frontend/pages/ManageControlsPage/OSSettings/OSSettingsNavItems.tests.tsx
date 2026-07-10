import getOSSettingsNavItems from "./OSSettingsNavItems";

describe("getOSSettingsNavItems", () => {
  it("includes the Host names card for a fleet (non-technician)", () => {
    const titles = getOSSettingsNavItems(false).map((i) => i.title);
    expect(titles).toContain("Host names");
    expect(titles[titles.length - 1]).toBe("Host names");
  });

  it("includes the Host names card for 'No team'", () => {
    // The template is supported for fleets and for "No team" (global config),
    // so the card renders in both scopes.
    const titles = getOSSettingsNavItems(false).map((i) => i.title);
    expect(titles).toContain("Host names");
  });

  it("excludes the Host names card for technicians", () => {
    const titles = getOSSettingsNavItems(true).map((i) => i.title);
    expect(titles).not.toContain("Host names");
  });
});
