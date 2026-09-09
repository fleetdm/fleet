import React from "react";
import { noop } from "lodash";
import { render, screen } from "@testing-library/react";

import { createMockRouter } from "test/test-utils";

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

  // installed_versions[] entries carry no source of their own, so the cell has
  // to read the parent row's. rpm_packages populates release too, and must keep
  // rendering the plain version.
  it("renders the Installed version column with the row's source applied", () => {
    const versionColumn = headers.find((h) => h.id === "version") as
      | { Cell?: React.ElementType }
      | undefined;
    const Cell = versionColumn?.Cell as React.ElementType;

    const goRow = render(
      <Cell
        cell={{ value: [{ version: "v0.21.1", release: "go1.26.1" }] }}
        row={{ original: { source: "go_binaries" } }}
      />
    );
    expect(screen.getAllByText("v0.21.1 (go1.26.1)")[0]).toBeInTheDocument();
    goRow.unmount();

    render(
      <Cell
        cell={{ value: [{ version: "1.2.3", release: "30.el7" }] }}
        row={{ original: { source: "rpm_packages" } }}
      />
    );
    expect(screen.getAllByText("1.2.3")[0]).toBeInTheDocument();
  });
});
