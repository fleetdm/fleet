import { render, screen, waitFor } from "@testing-library/react";
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

    it.each([
      {
        source: "mcp_servers",
        tip:
          "MCP servers don't have versions. They update on the fly, locally or remotely.",
      },
      {
        source: "ai_skills",
        tip: "AI skills are markdown files, so they don't have versions.",
      },
    ] as const)(
      "explains the empty value for $source on hover",
      async ({ source, tip }) => {
        const { user } = renderWithSetup(
          <Cell
            row={{
              original: createMockSoftwareVersion({ source, version: "" }),
            }}
          />
        );

        await user.hover(screen.getByText(DEFAULT_EMPTY_CELL_VALUE));

        await waitFor(() => {
          expect(screen.getByText(tip)).toBeInTheDocument();
        });
      }
    );

    it("renders an AI CLI tool's version", () => {
      render(
        <Cell
          row={{
            original: createMockSoftwareVersion({
              source: "ai_clis",
              version: "2.0.14",
            }),
          }}
        />
      );

      expect(screen.getByText("2.0.14")).toBeInTheDocument();
    });
  });

  describe("Vulnerabilities column", () => {
    const Cell = getCell("Vulnerabilities");

    it.each([
      {
        source: "mcp_servers",
        tip: "Currently, Fleet doesn't detect vulnerabilities for MCP servers.",
      },
      {
        source: "ai_skills",
        tip:
          "AI skills are markdown files, so they don't have vulnerabilities.",
      },
    ] as const)(
      "explains the empty value for $source on hover",
      async ({ source, tip }) => {
        const { user } = renderWithSetup(
          <Cell
            cell={{ value: [] }}
            row={{
              original: createMockSoftwareVersion({
                source,
                version: "",
                vulnerabilities: [],
              }),
            }}
          />
        );

        await user.hover(screen.getByText(DEFAULT_EMPTY_CELL_VALUE));

        await waitFor(() => {
          expect(screen.getByText(tip)).toBeInTheDocument();
        });
      }
    );
  });
});
