import { isValidNumber, getVulnerabilities } from "./helpers";

describe("isValidNumber", () => {
  // Test valid numbers
  it("returns true for valid numbers", () => {
    expect(isValidNumber(0)).toBe(true);
    expect(isValidNumber(42)).toBe(true);
    expect(isValidNumber(-10)).toBe(true);
    expect(isValidNumber(3.14)).toBe(true);
  });

  // Test invalid inputs
  it("returns false for non-number inputs", () => {
    expect(isValidNumber("42")).toBe(false);
    expect(isValidNumber(null)).toBe(false);
    expect(isValidNumber(undefined)).toBe(false);
    expect(isValidNumber({})).toBe(false);
    expect(isValidNumber([])).toBe(false);
    expect(isValidNumber(true)).toBe(false);
  });

  // Test NaN
  it("returns false for NaN", () => {
    expect(isValidNumber(NaN)).toBe(false);
  });

  // Test with min value
  it("respects min value when provided", () => {
    expect(isValidNumber(5, 0)).toBe(true);
    expect(isValidNumber(5, 5)).toBe(true);
    expect(isValidNumber(5, 6)).toBe(false);
  });

  // Test with max value
  it("respects max value when provided", () => {
    expect(isValidNumber(5, undefined, 10)).toBe(true);
    expect(isValidNumber(5, undefined, 5)).toBe(true);
    expect(isValidNumber(5, undefined, 4)).toBe(false);
  });

  // Test with both min and max values
  it("respects both min and max values when provided", () => {
    expect(isValidNumber(5, 0, 10)).toBe(true);
    expect(isValidNumber(0, 0, 10)).toBe(true);
    expect(isValidNumber(10, 0, 10)).toBe(true);
    expect(isValidNumber(-1, 0, 10)).toBe(false);
    expect(isValidNumber(11, 0, 10)).toBe(false);
  });
});

const versions = [
  {
    id: 531270,
    version: "131.0.6778.86",
    vulnerabilities: ["CVE-2024-12053", "CVE-2024-12381", "CVE-2025-0444"],
  },
  {
    id: 538184,
    version: "132.0.6834.160",
    vulnerabilities: ["CVE-2025-0444", "CVE-2025-0445"], // 0444 is duplicate
  },
  {
    id: 541233,
    version: "133.0.6943.53",
    vulnerabilities: ["CVE-2025-0995", "CVE-2025-0996"],
  },
  {
    id: 572993,
    version: "139.0.7258.127",
    vulnerabilities: null, // should be ignored
  },
];

describe("getVulnerabilities", () => {
  it("returns a unique list of vulnerabilities across all versions", () => {
    const result = getVulnerabilities(versions);

    // Expect no duplicates
    expect(new Set(result).size).toBe(result.length);

    // Expect specific vulns present
    expect(result).toEqual(
      expect.arrayContaining([
        "CVE-2024-12053",
        "CVE-2024-12381",
        "CVE-2025-0444",
        "CVE-2025-0445",
        "CVE-2025-0995",
        "CVE-2025-0996",
      ])
    );

    // Should not contain unintended values
    expect(result).not.toContain("CVE-DOES-NOT-EXIST");
  });

  it("returns an empty array if no versions are given", () => {
    expect(getVulnerabilities([])).toEqual([]);
  });
});
