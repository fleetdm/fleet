import { screen, waitFor } from "@testing-library/react";
import React from "react";

import { createMockHostSoftware } from "__mocks__/hostMock";
import { renderWithSetup } from "test/test-utils";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

import { generateSoftwareTableHeaders } from "./DeviceSoftwareTableConfig";

describe("DeviceSoftwareTableConfig", () => {
  const headers = generateSoftwareTableHeaders();

  describe("Vulnerabilities column", () => {
    const vulnColumn = headers.find(
      (h) => typeof h.Header === "string" && h.Header === "Vulnerabilities"
    ) as { Cell?: React.ElementType } | undefined;
    const Cell = vulnColumn?.Cell as React.ElementType;

    // The tooltip copy per source is covered in SoftwareInventoryTableConfig.
    it("explains the empty value for MCP servers on hover", async () => {
      const { user } = renderWithSetup(
        <Cell
          cell={{ value: [{ version: "", vulnerabilities: [] }] }}
          row={{ original: createMockHostSoftware({ source: "mcp_servers" }) }}
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
