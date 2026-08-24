import { SeverityValue } from "components/SeverityFilter";
import { ALL_CVE_SOFTWARE_CATEGORY_VALUES } from "interfaces/charts";

import {
  buildInitialChartFilters,
  hostFilterLines,
  softwareFilterLines,
} from "./helpers";

describe("buildInitialChartFilters", () => {
  it("uses built-in defaults when no persisted defaults are provided", () => {
    const filters = buildInitialChartFilters(undefined);
    expect(filters.softwareFilters).toEqual([
      ...ALL_CVE_SOFTWARE_CATEGORY_VALUES,
    ]);
    expect(filters.knownExploit).toBe(false);
    expect(filters.epssMin).toBe("");
    expect(filters.epssMax).toBe("");
    expect(filters.severity).toBe("critical");
    expect(filters.cvssMin).toBe("9");
    expect(filters.cvssMax).toBe("10");
    expect(filters.excludeCVEs).toEqual([]);
  });

  it("keeps the critical severity default when no CVSS bounds are persisted", () => {
    const filters = buildInitialChartFilters({ has_known_exploit: true });
    expect(filters.severity).toBe("critical");
    expect(filters.cvssMin).toBe("9");
    expect(filters.cvssMax).toBe("10");
  });

  it("seeds a preset with its own bounds, which is what gets sent to the API", () => {
    expect(
      buildInitialChartFilters({ cvss_min: 7, cvss_max: 8.9 })
    ).toMatchObject({ severity: "high", cvssMin: "7", cvssMax: "8.9" });
    // Any narrows nothing, so it carries no bounds.
    expect(
      buildInitialChartFilters({ cvss_min: 0, cvss_max: 10 })
    ).toMatchObject({ severity: "any", cvssMin: "", cvssMax: "" });
  });

  it("seeds a custom range when only one CVSS bound is persisted", () => {
    expect(buildInitialChartFilters({ cvss_min: 7 })).toMatchObject({
      severity: "custom",
      cvssMin: "7",
      cvssMax: "10",
    });
  });

  it("seeds present fields and falls back per-field for absent ones", () => {
    const filters = buildInitialChartFilters({
      software_filters: ["browsers"],
      has_known_exploit: true,
    });
    expect(filters.softwareFilters).toEqual(["browsers"]);
    expect(filters.knownExploit).toBe(true);
    expect(filters.epssMin).toBe("");
    expect(filters.epssMax).toBe("");
    expect(filters.excludeCVEs).toEqual([]);
  });

  it("converts numeric EPSS bounds (0-100) to strings", () => {
    const filters = buildInitialChartFilters({ epss_min: 0, epss_max: 90 });
    expect(filters.epssMin).toBe("0");
    expect(filters.epssMax).toBe("90");
  });

  it("honors an explicit empty software_filters list as 'none'", () => {
    const filters = buildInitialChartFilters({ software_filters: [] });
    expect(filters.softwareFilters).toEqual([]);
  });

  it("seeds the exclude-CVE list", () => {
    const filters = buildInitialChartFilters({
      exclude_vulnerabilities: ["CVE-2025-50897"],
    });
    expect(filters.excludeCVEs).toEqual(["CVE-2025-50897"]);
  });
});

describe("softwareFilterLines", () => {
  // The bounds always mirror the selection, so a preset here is given its own —
  // the same state the app can actually produce.
  const filtersWithSeverity = (
    severity: SeverityValue,
    cvssMin = "",
    cvssMax = ""
  ) => ({
    ...buildInitialChartFilters(undefined),
    severity,
    cvssMin,
    cvssMax,
  });

  // How a selection reads is severityValueLabel's contract, covered by its own
  // tests. What belongs here is the line: its prefix, and the gate deciding
  // whether there is one at all.
  it("gives the active severity a prefixed line of its own", () => {
    expect(
      softwareFilterLines(filtersWithSeverity("critical", "9", "10"))
    ).toEqual(["Severity: Critical"]);
    expect(
      softwareFilterLines(filtersWithSeverity("custom", "2.5", "6"))
    ).toEqual(["Severity: Custom (2.5 to 6)"]);
  });

  it("omits the severity line for Any severity", () => {
    expect(softwareFilterLines(filtersWithSeverity("any"))).toEqual([]);
  });

  it("omits the severity line for a Custom range that narrows nothing", () => {
    // Nothing is sent to the API in either state, so claiming a severity filter
    // would describe the chart as narrower than it is.
    expect(softwareFilterLines(filtersWithSeverity("custom"))).toEqual([]);
    // Typing both ends of the scale into Custom is the same as not filtering,
    // even though the dropdown still reads Custom — typing never re-derives it.
    expect(
      softwareFilterLines(filtersWithSeverity("custom", "0", "10"))
    ).toEqual([]);
    // A single bound is enough to be a real filter.
    expect(
      softwareFilterLines(filtersWithSeverity("custom", "", "6"))
    ).toEqual(["Severity: Custom (0 to 6)"]);
  });

  it("keeps severity separate from the generic Advanced filters line", () => {
    expect(
      softwareFilterLines({
        ...filtersWithSeverity("high", "7", "8.9"),
        excludeCVEs: ["CVE-2025-0001"],
      })
    ).toEqual(["Severity: High", "Advanced filters"]);
  });
});

describe("hostFilterLines", () => {
  const filtersWithPlatforms = (platforms: string[]) => ({
    ...buildInitialChartFilters(undefined),
    platforms,
  });

  it("preserves branded platform casing (macOS, iOS, iPadOS)", () => {
    const [line] = hostFilterLines(
      filtersWithPlatforms(["darwin", "ios", "ipados"])
    );
    expect(line).toBe("macOS, iOS, and iPadOS");
    // Guards the reported bug: no word-capitalized variants.
    expect(line).not.toMatch(/MacOS|Ios|Ipados/);
  });

  it("renders a single platform without mangling its casing", () => {
    expect(hostFilterLines(filtersWithPlatforms(["darwin"]))).toEqual([
      "macOS",
    ]);
  });

  it("maps every filterable platform to its correct display name", () => {
    const [line] = hostFilterLines(
      filtersWithPlatforms([
        "darwin",
        "windows",
        "linux",
        "chrome",
        "ios",
        "ipados",
        "android",
      ])
    );
    expect(line).toBe(
      "macOS, Windows, Linux, ChromeOS, iOS, iPadOS, and Android"
    );
  });
});
