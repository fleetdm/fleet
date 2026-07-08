import getOSSettingsNavItems from "./OSSettingsNavItems";

describe("getOSSettingsNavItems", () => {
  it("includes the Host names card for a fleet (non-technician)", () => {
    const titles = getOSSettingsNavItems(false, false).map((i) => i.title);
    expect(titles).toContain("Host names");
    expect(titles[titles.length - 1]).toBe("Host names");
  });

  it("excludes the Host names card for 'No team' (fleets-only)", () => {
    const titles = getOSSettingsNavItems(false, true).map((i) => i.title);
    expect(titles).not.toContain("Host names");
  });

  it("excludes the Host names card for technicians", () => {
    const titles = getOSSettingsNavItems(true, false).map((i) => i.title);
    expect(titles).not.toContain("Host names");
  });
});
