import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import InstallerStatusTable from "./InstallerStatusTable";

describe("InstallerStatusTable", () => {
  const render = createCustomRenderer();

  it("renders columns and links for statuses", () => {
    render(
      <InstallerStatusTable
        softwareId={123}
        teamId={5}
        status={{ installed: 0, pending: 1, failed: 3 }}
      />
    );

    // Check cell values (always "hosts", even for 1)
    const cells = screen.getAllByRole("cell");
    expect(cells[0]).toHaveTextContent("0 hosts");
    expect(cells[1]).toHaveTextContent("1 host");
    expect(cells[2]).toHaveTextContent("3 hosts");

    // Check the anchor and its text in each cell
    expect(cells[0].querySelector("a.link-cell")).toHaveTextContent("0 hosts");
    expect(cells[1].querySelector("a.link-cell")).toHaveTextContent("1 host");
    expect(cells[2].querySelector("a.link-cell")).toHaveTextContent("3 hosts");
  });
});
