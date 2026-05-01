import { createMockRouter } from "test/test-utils";

import generateTableHeaders from "./SoftwareInventoryTableConfig";

const mockRouter = createMockRouter();

describe("SoftwareInventoryTableConfig", () => {
  const headers = generateTableHeaders(mockRouter, 1);

  it("generates the correct column headers", () => {
    const headerNames = headers.map((h) => {
      if (typeof h.Header === "string") return h.Header;
      return h.accessor; // sortable headers use accessor as key
    });

    expect(headerNames).toEqual([
      "name",
      "Version",
      "Type",
      "Vulnerabilities",
      "hosts_count",
      "",
    ]);
  });

  it("has a Version column (not 'Installed version')", () => {
    const versionCol = headers.find(
      (h) => typeof h.Header === "string" && h.Header === "Version"
    );
    expect(versionCol).toBeDefined();
  });

  it("does not have a Library version column", () => {
    const libraryVersionCol = headers.find(
      (h) => typeof h.Header === "string" && h.Header === "Library version"
    );
    expect(libraryVersionCol).toBeUndefined();
  });

  it("has a Vulnerabilities column", () => {
    const vulnCol = headers.find(
      (h) => typeof h.Header === "string" && h.Header === "Vulnerabilities"
    );
    expect(vulnCol).toBeDefined();
  });

  it("disables sorting on Version, Type, and Vulnerabilities", () => {
    const nonSortable = headers.filter((h) => h.disableSortBy === true);
    const nonSortableKeys = nonSortable.map(
      (h) => (typeof h.Header === "string" ? h.Header : h.id) || h.accessor
    );

    expect(nonSortableKeys).toContain("Version");
    expect(nonSortableKeys).toContain("Type");
    expect(nonSortableKeys).toContain("Vulnerabilities");
  });
});
