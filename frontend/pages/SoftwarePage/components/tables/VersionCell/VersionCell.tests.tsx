import { render, screen, waitFor } from "@testing-library/react";
import React from "react";

import { createMockHostSoftware } from "__mocks__/hostMock";
import { renderWithSetup } from "test/test-utils";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

import VersionCell, { VersionsColumnCell } from "./VersionCell";

describe("VersionCell", () => {
  it("renders the empty value when there are no versions", () => {
    render(<VersionCell versions={null} source="apps" />);

    expect(screen.getByText(DEFAULT_EMPTY_CELL_VALUE)).toBeInTheDocument();
  });

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
