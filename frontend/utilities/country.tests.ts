import { formatCountry } from "./country";

describe("formatCountry", () => {
  it("formats a lowercase code as flag and name", () => {
    expect(formatCountry("jp")).toBe("🇯🇵 Japan");
  });

  it("formats an uppercase code as flag and name", () => {
    expect(formatCountry("US")).toBe("🇺🇸 United States");
  });

  it("returns an empty string for a missing code", () => {
    expect(formatCountry("")).toBe("");
    expect(formatCountry(null)).toBe("");
    expect(formatCountry(undefined)).toBe("");
  });

  it("falls back to the uppercased code when it isn't two letters", () => {
    expect(formatCountry("usa")).toBe("USA");
  });
});
