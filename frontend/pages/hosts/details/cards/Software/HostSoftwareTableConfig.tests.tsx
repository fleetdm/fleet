import { render, screen, waitFor } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import { createMockHostSoftware } from "__mocks__/hostMock";
import { createMockRouter, renderWithSetup } from "test/test-utils";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

import { generateSoftwareTableHeaders } from "./HostSoftwareTableConfig";

const mockRouter = createMockRouter();

describe("HostSoftwareTableConfig", () => {
  const headers = generateSoftwareTableHeaders({
    router: mockRouter,
    teamId: 1,
    onShowInventoryVersions: noop,
  });

  describe("Last opened column", () => {
    const lastOpenedColumn = headers.find((h) => h.id === "Last opened") as any;

    if (!lastOpenedColumn || typeof lastOpenedColumn.accessor !== "function") {
      throw new Error("Last opened column or accessor not found");
    }

    const Cell = lastOpenedColumn.Cell as React.ElementType;

    it("renders the date when a valid date string is provided", () => {
      render(
        <Cell cell={{ value: "2023-01-01T00:00:00Z" }} row={{ original: {} }} />
      );
      // HumanTimeDiffWithDateTip will render something like "2 years ago" or similar depending on current date
      // but it definitely won't be "Never" or "Not supported"
      expect(screen.queryByText("Never")).not.toBeInTheDocument();
      expect(screen.queryByText("Not supported")).not.toBeInTheDocument();
    });

    it("renders 'Never' when the value is an empty string", () => {
      render(<Cell cell={{ value: "" }} row={{ original: {} }} />);
      expect(screen.getByText("Never")).toBeInTheDocument();
    });

    it("renders 'Not supported' when the value is undefined", () => {
      render(<Cell cell={{ value: undefined }} row={{ original: {} }} />);
      expect(screen.getByText("Not supported")).toBeInTheDocument();
    });
  });

  // installed_versions[] entries carry no source; the cell must read the row's.
  describe("Installed version column", () => {
    const versionColumn = headers.find((h) => h.id === "version") as
      | { Cell?: React.ElementType }
      | undefined;
    const Cell = versionColumn?.Cell as React.ElementType;

    it("appends the Go toolchain version for a go_binaries row", () => {
      render(
        <Cell
          cell={{ value: [{ version: "v0.21.1", release: "go1.26.1" }] }}
          row={{ original: createMockHostSoftware({ source: "go_binaries" }) }}
        />
      );

      expect(screen.getAllByText("v0.21.1 (go1.26.1)")[0]).toBeInTheDocument();
    });

    it("renders the plain version for a source that also populates release", () => {
      render(
        <Cell
          cell={{ value: [{ version: "1.2.3", release: "30.el7" }] }}
          row={{ original: createMockHostSoftware({ source: "rpm_packages" }) }}
        />
      );

      expect(screen.getAllByText("1.2.3")[0]).toBeInTheDocument();
    });
  });

  describe("AI tool rows without versions or vulnerabilities", () => {
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
        column: "version",
        source: "ai_skills",
        tip: "AI skills are markdown files, so they don't have versions.",
      },
      {
        column: "Vulnerabilities",
        source: "mcp_servers",
        tip: "Currently, Fleet doesn't detect vulnerabilities for MCP servers.",
      },
      {
        column: "Vulnerabilities",
        source: "ai_skills",
        tip:
          "AI skills are markdown files, so they don't have vulnerabilities.",
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
              installed_paths: ["/Users/alice/.claude/settings.json"],
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
});
