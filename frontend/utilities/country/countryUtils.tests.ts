import { getCountryDisplay, getCountryFlag } from "./countryUtils";

describe("countryUtils", () => {
  describe("getCountryFlag", () => {
    it("returns the flag emoji for a lowercase country code", () => {
      expect(getCountryFlag("jp")).toBe("🇯🇵");
      expect(getCountryFlag("us")).toBe("🇺🇸");
    });
  });

  describe("getCountryDisplay", () => {
    it("returns the flag and country name", () => {
      expect(getCountryDisplay("jp")).toBe("🇯🇵 Japan");
      expect(getCountryDisplay("US")).toBe("🇺🇸 United States");
    });

    it("returns undefined for an empty country code", () => {
      expect(getCountryDisplay("")).toBeUndefined();
      expect(getCountryDisplay(undefined)).toBeUndefined();
    });

    it("falls back to the uppercase code when the country is unknown", () => {
      expect(getCountryDisplay("xx")).toBe("XX");
      expect(getCountryDisplay("usa")).toBe("USA");
    });
  });
});
