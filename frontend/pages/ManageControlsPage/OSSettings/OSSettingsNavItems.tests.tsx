import getOSSettingsNavItems from "./OSSettingsNavItems";

describe("getOSSettingsNavItems", () => {
  it("includes the Host names card, last, for non-technicians", () => {
    // The card is team-agnostic in the nav — it renders for both fleets and
    // "No team"; scope is resolved inside the card, not by the nav filter.
    const titles = getOSSettingsNavItems(false).map((i) => i.title);
    expect(titles).toContain("Host names");
    expect(titles[titles.length - 1]).toBe("Host names");
  });

  it("excludes the Host names card for technicians", () => {
    const titles = getOSSettingsNavItems(true).map((i) => i.title);
    expect(titles).not.toContain("Host names");
  });
});
