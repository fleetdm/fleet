import { getVulnerabilities } from "./helpers";

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
