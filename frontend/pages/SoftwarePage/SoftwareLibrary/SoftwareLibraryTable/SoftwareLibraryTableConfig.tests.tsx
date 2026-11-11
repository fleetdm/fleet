import { screen } from "@testing-library/react";
import React from "react";

import {
  createMockSoftwarePackage,
  createMockSoftwareTitle,
} from "__mocks__/softwareMock";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import generateTableHeaders from "./SoftwareLibraryTableConfig";

const mockRouter = createMockRouter();

describe("SoftwareLibraryTableConfig", () => {
  const headers = generateTableHeaders(mockRouter, 1);

  it("generates the correct column headers", () => {
    const headerNames = headers.map((h) => {
      if (typeof h.Header === "string") return h.Header;
      return h.accessor;
    });

    expect(headerNames).toEqual([
      "name",
      "Installed version",
      "Library version",
      "Type",
      "hosts_count",
      "",
    ]);
  });

  it("has 'Installed version' instead of 'Version'", () => {
    const installedVersionCol = headers.find(
      (h) => typeof h.Header === "string" && h.Header === "Installed version"
    );
    const plainVersionCol = headers.find(
      (h) => typeof h.Header === "string" && h.Header === "Version"
    );

    expect(installedVersionCol).toBeDefined();
    expect(plainVersionCol).toBeUndefined();
  });

  it("has a Library version column", () => {
    const libraryVersionCol = headers.find((h) => h.id === "library_version");
    expect(libraryVersionCol).toBeDefined();
  });

  it("does not have a Vulnerabilities column", () => {
    const vulnCol = headers.find(
      (h) => typeof h.Header === "string" && h.Header === "Vulnerabilities"
    );
    expect(vulnCol).toBeUndefined();
  });

  it("disables sorting on Installed version, Library version, and Type", () => {
    const nonSortable = headers.filter((h) => h.disableSortBy === true);
    const nonSortableKeys = nonSortable.map(
      (h) => (typeof h.Header === "string" ? h.Header : h.id) || h.accessor
    );

    expect(nonSortableKeys).toContain("Installed version");
    expect(nonSortableKeys).toContain("Library version");
    expect(nonSortableKeys).toContain("Type");
  });

  it("shows the automatic install indicator when only a later package has automatic_install_policies", () => {
    const title = createMockSoftwareTitle({
      app_store_app: null,
      packages: [
        createMockSoftwarePackage({
          installer_id: 1,
          automatic_install_policies: null,
        }),
        createMockSoftwarePackage({
          installer_id: 2,
          automatic_install_policies: [
            { id: 1, name: "Install it", type: "dynamic" },
          ],
        }),
      ],
    });
    const nameColumn = headers.find((h) => h.accessor === "name");
    const NameCell = nameColumn?.Cell as React.ComponentType<{
      row: { original: typeof title };
    }>;
    const render = createCustomRenderer({ withBackendMock: true });

    render(<NameCell row={{ original: title }} />);

    expect(screen.getByTestId("install-auto-icon")).toBeInTheDocument();
  });
});
