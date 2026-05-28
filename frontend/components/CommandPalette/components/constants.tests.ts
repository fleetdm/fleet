import { RESULT_PREFIXES, isPreFilteredResult } from "./constants";

describe("isPreFilteredResult", () => {
  it("returns true for every value beginning with a known RESULT_PREFIX", () => {
    Object.values(RESULT_PREFIXES).forEach((prefix) => {
      expect(isPreFilteredResult(`${prefix}42`)).toBe(true);
    });
  });

  it("returns false for arbitrary strings", () => {
    expect(isPreFilteredResult("dashboard home")).toBe(false);
    expect(isPreFilteredResult("EXACT_MATCH dashboard")).toBe(false);
    expect(isPreFilteredResult("")).toBe(false);
  });

  it("returns false for the singular form (typo guard)", () => {
    // Catches a future picker typoing the prefix (e.g., "HOSTS_RESULT ").
    expect(isPreFilteredResult("HOSTS_RESULT 1")).toBe(false);
    expect(isPreFilteredResult("HOST_RESULTS 1")).toBe(false);
    expect(isPreFilteredResult("REPORTS_RESULT 1")).toBe(false);
  });
});
