import { getVulnerabilities, getVulnFilterRenderDetails } from "./helpers";

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

describe("getVulnFilterRenderDetails", () => {
  // Only the CVSS bounds are stored, so the severity line is recovered from
  // them by the same helpers the chart's filter summary uses.
  const withSeverity = (minCvssScore?: number, maxCvssScore?: number) => {
    const details = getVulnFilterRenderDetails({
      vulnerable: true,
      minCvssScore,
      maxCvssScore,
    });
    return {
      filterCount: details.filterCount,
      // Each entry is a <span>{line}<br /></span> keyed by uniqueId, so the
      // text is the first child.
      lines: details.tooltipText.map((line) => line.props.children[0]),
    };
  };

  it("counts nothing but the vulnerable filter when no scores are set", () => {
    expect(withSeverity()).toEqual({
      filterCount: 1,
      lines: ["Vulnerable software"],
    });
  });

  it("names a preset without restating its band", () => {
    expect(withSeverity(9, 10).lines).toContain("Severity: Critical");
    expect(withSeverity(0.1, 3.9).lines).toContain("Severity: Low");
  });

  it("spells out a Custom range, widening a half-open one", () => {
    expect(withSeverity(2.5, 6).lines).toContain("Severity: Custom (2.5 to 6)");
    expect(withSeverity(2.5, undefined).lines).toContain(
      "Severity: Custom (2.5 to 10)"
    );
  });

  it("counts a lone 0 minimum, which is a real bound", () => {
    // The modal submits min 0 with no max, so this reaches the URL through the
    // UI. A truthiness test here dropped it and under-reported the count.
    const { filterCount, lines } = withSeverity(0, undefined);
    expect(filterCount).toBe(2);
    expect(lines).toContain("Severity: Custom (0 to 10)");
  });

  it("does not count the whole 0-10 scale as a severity filter", () => {
    // It spans the scale, so it narrows nothing.
    const { filterCount, lines } = withSeverity(0, 10);
    expect(filterCount).toBe(1);
    expect(lines).not.toContainEqual(expect.stringContaining("Severity"));
  });

  it("ignores severity entirely when the vulnerable filter is off", () => {
    const { filterCount, buttonText } = getVulnFilterRenderDetails({
      vulnerable: false,
      minCvssScore: 9,
      maxCvssScore: 10,
    });
    expect(filterCount).toBe(0);
    expect(buttonText).toBe("Add filters");
  });
});
