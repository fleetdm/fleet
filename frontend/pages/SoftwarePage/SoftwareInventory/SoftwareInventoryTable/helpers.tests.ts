import { mapValues } from "lodash";

import {
  buildSoftwareFiltersQueryParams,
  getFilterRenderDetails,
  getSoftwareFiltersFromQueryParams,
  getVulnerabilities,
  removePremiumOnlyFilters,
} from "./helpers";

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

describe("getSoftwareFiltersFromQueryParams", () => {
  it("parses types through the catalog", () => {
    expect(
      getSoftwareFiltersFromQueryParams({ types: "foo,macos_app" }).types
    ).toEqual(["macos_app"]);
  });

  it("reads the AI tools filter", () => {
    expect(getSoftwareFiltersFromQueryParams({ ai_tool: "true" }).aiTool).toBe(
      true
    );
    expect(getSoftwareFiltersFromQueryParams({}).aiTool).toBe(false);
  });

  it.each([
    { name: "alone", filters: { aiTool: true } },
    {
      name: "with vulnerable off",
      filters: { aiTool: true, vulnerable: false, types: ["mcp_server"] },
    },
    {
      name: "with vulnerable on",
      filters: { aiTool: true, vulnerable: true, exploit: true },
    },
  ])("round trips the AI tools filter $name", ({ filters }) => {
    const params = buildSoftwareFiltersQueryParams(filters);
    expect(params.ai_tool).toBe(true);
    expect(
      // URL values arrive as strings.
      getSoftwareFiltersFromQueryParams(mapValues(params, String))
    ).toEqual(expect.objectContaining(filters));
  });
});

describe("buildSoftwareFiltersQueryParams", () => {
  it("emits sorted types even when the vulnerable filter is off", () => {
    expect(
      buildSoftwareFiltersQueryParams({
        vulnerable: false,
        types: ["macos_app", "cursor_extension", "brave_extension"],
      })
    ).toEqual({ types: "brave_extension,cursor_extension,macos_app" });
  });

  it("emits types alongside the vulnerability params", () => {
    expect(
      buildSoftwareFiltersQueryParams({
        vulnerable: true,
        exploit: true,
        types: ["macos_app"],
      })
    ).toEqual({ types: "macos_app", vulnerable: true, exploit: true });
  });

  it("emits ai_tool when the vulnerable filter is off", () => {
    expect(
      buildSoftwareFiltersQueryParams({ vulnerable: false, aiTool: true })
    ).toEqual({ ai_tool: true });
  });

  it("omits ai_tool when the AI tools filter is off", () => {
    expect(
      buildSoftwareFiltersQueryParams({ vulnerable: true, aiTool: false })
    ).toEqual({ vulnerable: true });
  });

  it("omits types when none are selected", () => {
    expect(
      buildSoftwareFiltersQueryParams({ vulnerable: false, types: [] })
    ).toEqual({});
  });
});

describe("getFilterRenderDetails", () => {
  it.each([
    {
      name: "nothing",
      filters: { vulnerable: false, types: [] },
      isFiltered: false,
    },
    {
      name: "only types",
      filters: { vulnerable: false, types: ["macos_app"] },
      isFiltered: true,
    },
    {
      name: "only vulnerable",
      filters: { vulnerable: true },
      isFiltered: true,
    },
    {
      name: "only AI tools",
      filters: { vulnerable: false, aiTool: true },
      isFiltered: true,
    },
  ])("is filtered by $name: $isFiltered", ({ filters, isFiltered }) => {
    expect(getFilterRenderDetails(filters)).toEqual({
      isFiltered,
      buttonText: isFiltered ? "Filtered" : "Add filters",
    });
  });
});

describe("removePremiumOnlyFilters", () => {
  it("drops the AI tools filter and Premium-only types", () => {
    expect(
      removePremiumOnlyFilters({
        vulnerable: true,
        aiTool: true,
        types: ["mcp_server", "macos_app", "ai_skill", "ai_cli_tool"],
      })
    ).toEqual({ vulnerable: true, aiTool: false, types: ["macos_app"] });
  });

  it("keeps an absent types param absent", () => {
    expect(removePremiumOnlyFilters({ aiTool: true }).types).toBeUndefined();
  });
});
