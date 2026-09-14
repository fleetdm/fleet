import React from "react";
import { render, screen } from "@testing-library/react";

import { createMockRouter } from "test/test-utils";
import {
  createMockSoftwareTitle,
  createMockSoftwareTitleVersion,
} from "__mocks__/softwareMock";

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

  // Versions carry no source of their own, so the cell reads the title's.
  describe("Version column", () => {
    const versionCol = headers.find(
      (h) => typeof h.Header === "string" && h.Header === "Version"
    ) as { Cell?: React.ElementType } | undefined;
    const Cell = versionCol?.Cell as React.ElementType;

    it("appends the Go toolchain version for a go_binaries title", () => {
      render(
        <Cell
          cell={{
            value: [
              createMockSoftwareTitleVersion({
                version: "v0.21.1",
                release: "go1.26.1",
              }),
            ],
          }}
          row={{ original: createMockSoftwareTitle({ source: "go_binaries" }) }}
        />
      );

      expect(screen.getAllByText("v0.21.1 (go1.26.1)")[0]).toBeInTheDocument();
    });

    it("renders the plain version for a source that also populates release", () => {
      render(
        <Cell
          cell={{
            value: [
              createMockSoftwareTitleVersion({
                version: "1.2.3",
                release: "30.el7",
              }),
            ],
          }}
          row={{
            original: createMockSoftwareTitle({ source: "rpm_packages" }),
          }}
        />
      );

      expect(screen.getAllByText("1.2.3")[0]).toBeInTheDocument();
    });
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
