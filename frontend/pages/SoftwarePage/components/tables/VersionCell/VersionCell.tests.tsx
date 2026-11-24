import { render, screen, waitFor } from "@testing-library/react";
import React from "react";

import { createMockHostSoftware } from "__mocks__/hostMock";
import { renderWithSetup } from "test/test-utils";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

import VersionCell, { VersionsColumnCell } from "./VersionCell";

describe("VersionCell", () => {
  it.each(["apps", "ai_clis"] as const)(
    "renders the empty value without a tooltip for %s",
    async (source) => {
      const { user } = renderWithSetup(
        <VersionCell versions={null} source={source} />
      );

      await user.hover(screen.getByText(DEFAULT_EMPTY_CELL_VALUE));

      expect(screen.queryByRole("tooltip")).not.toBeInTheDocument();
    }
  );

  it.each([
    {
      source: "mcp_servers",
      tip:
        "Fleet doesn't detect MCP server versions yet. Most MCP servers run with npx or uvx, which pick the version at launch unless it's pinned.",
    },
    {
      source: "ai_skills",
      tip: "AI skills are markdown files, so they don't have versions.",
    },
  ] as const)(
    "explains the empty value for $source on hover",
    async ({ source, tip }) => {
      const { user } = renderWithSetup(
        <VersionCell versions={[{ version: "" }]} source={source} />
      );

      await user.hover(screen.getByText(DEFAULT_EMPTY_CELL_VALUE));

      await waitFor(() => {
        expect(screen.getByText(tip)).toBeInTheDocument();
      });
    }
  );

  it("renders a single version", () => {
    render(<VersionCell versions={[{ version: "1.2.3" }]} source="apps" />);

    expect(screen.getAllByText("1.2.3")[0]).toBeInTheDocument();
  });

  it("renders a count with every version in the tooltip for multiple versions", async () => {
    const { user } = renderWithSetup(
      <VersionCell
        versions={[
          { version: "v0.21.1", release: "go1.26.1" },
          { version: "v0.21.1", release: "go1.25.4" },
        ]}
        source="go_binaries"
      />
    );

    await user.hover(screen.getByText("2 versions"));

    await waitFor(() => {
      expect(
        screen.getByText("v0.21.1 (go1.26.1), v0.21.1 (go1.25.4)")
      ).toBeInTheDocument();
    });
  });

  describe("VersionsColumnCell", () => {
    it("formats the cell's versions with the row's source", () => {
      const Cell = VersionsColumnCell as React.ElementType;
      render(
        <Cell
          cell={{ value: [{ version: "v0.21.1", release: "go1.26.1" }] }}
          row={{ original: createMockHostSoftware({ source: "go_binaries" }) }}
        />
      );

      expect(screen.getAllByText("v0.21.1 (go1.26.1)")[0]).toBeInTheDocument();
    });
  });
});
