import { screen, waitFor } from "@testing-library/react";
import React from "react";

import { createMockHostSoftware } from "__mocks__/hostMock";
import { renderWithSetup } from "test/test-utils";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

import { generateSoftwareTableHeaders } from "./DeviceSoftwareTableConfig";

describe("DeviceSoftwareTableConfig", () => {
  const headers = generateSoftwareTableHeaders();
  const cellFor = (column: string) =>
    (headers.find((h) => h.id === column || h.Header === column) as
      | { Cell?: React.ElementType }
      | undefined)?.Cell as React.ElementType;

  it.each([
    {
      column: "version",
      source: "mcp_servers",
      tip:
        "MCP servers don't have versions. They update on the fly, locally or remotely.",
    },
    {
      column: "Vulnerabilities",
      source: "mcp_servers",
      tip: "Currently, Fleet doesn't detect vulnerabilities for MCP servers.",
    },
    {
      column: "Vulnerabilities",
      source: "ai_skills",
      tip: "AI skills are markdown files, so they don't have vulnerabilities.",
    },
  ] as const)(
    "explains the empty $column cell for $source on hover",
    async ({ column, source, tip }) => {
      const original = createMockHostSoftware({
        source,
        installed_versions: [
          {
            version: "",
            bundle_identifier: "",
            vulnerabilities: null,
            installed_paths: ["/Users/alice/.claude/skills/review/SKILL.md"],
          },
        ],
      });
      const Cell = cellFor(column);
      const { user } = renderWithSetup(
        <Cell
          cell={{ value: original.installed_versions }}
          row={{ original }}
        />
      );

      await user.hover(screen.getByText(DEFAULT_EMPTY_CELL_VALUE));

      await waitFor(() => expect(screen.getByText(tip)).toBeInTheDocument());
    }
  );
});
