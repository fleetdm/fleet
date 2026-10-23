import { screen, waitFor } from "@testing-library/react";
import React from "react";

import { createMockSoftwareVersion } from "__mocks__/softwareMock";
import { createMockRouter, renderWithSetup } from "test/test-utils";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

import generateTableHeaders from "./SoftwareVersionsTableConfig";

const headers = generateTableHeaders(createMockRouter(), 1);

const getCell = (header: string) =>
  (headers.find((h) => h.Header === header) as {
    Cell: React.ElementType;
  }).Cell;

describe("SoftwareVersionsTableConfig", () => {
  describe("Version column", () => {
    const Cell = getCell("Version");

    // The tooltip copy per source is covered in VersionCell.
    it("explains the empty value for MCP servers on hover", async () => {
      const { user } = renderWithSetup(
        <Cell
          row={{
            original: createMockSoftwareVersion({
              source: "mcp_servers",
              version: "",
            }),
          }}
        />
      );

      await user.hover(screen.getByText(DEFAULT_EMPTY_CELL_VALUE));

      await waitFor(() => {
        expect(
          screen.getByText(
            "Fleet doesn't detect MCP server versions yet. Most MCP servers run with npx or uvx, which pick the version at launch unless it's pinned."
          )
        ).toBeInTheDocument();
      });
    });
  });

  describe("Vulnerabilities column", () => {
    const Cell = getCell("Vulnerabilities");

    // The tooltip copy per source is covered in SoftwareInventoryTableConfig.
    it("explains the empty value for MCP servers on hover", async () => {
      const { user } = renderWithSetup(
        <Cell
          cell={{ value: [] }}
          row={{
            original: createMockSoftwareVersion({
              source: "mcp_servers",
              version: "",
              vulnerabilities: [],
            }),
          }}
        />
      );

      await user.hover(screen.getByText(DEFAULT_EMPTY_CELL_VALUE));

      await waitFor(() => {
        expect(
          screen.getByText(
            "Currently, Fleet doesn't detect vulnerabilities for MCP servers."
          )
        ).toBeInTheDocument();
      });
    });
  });
});
