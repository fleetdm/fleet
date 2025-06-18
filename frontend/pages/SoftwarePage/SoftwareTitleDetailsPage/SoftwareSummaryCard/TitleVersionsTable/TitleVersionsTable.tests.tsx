import React from "react";
import { screen, render } from "@testing-library/react";
import { createMockRouter } from "test/test-utils";
import { ISoftwareTitleVersion } from "interfaces/software";
import TitleVersionsTable from "./TitleVersionsTable";

const mockRouter = createMockRouter;

describe("TitleVersionsTable", () => {
  it("renders version names as links and footer info", () => {
    const versions = [
      { id: 10, version: "1.2.3", vulnerabilities: [] },
      { id: 11, version: "1.2.4", vulnerabilities: [] },
    ];

    render(
      <TitleVersionsTable
        router={mockRouter}
        data={versions}
        isLoading={false}
        teamIdForApi={42}
        isIPadOSOrIOSApp={false}
        countsUpdatedAt="2024-05-08T12:00:00Z"
      />
    );

    // There should be one cell with a link for the version
    const cells = screen.getAllByRole("cell");
    expect(cells).toHaveLength(8);
    expect(screen.getByText(/1.2.3/i)).toBeInTheDocument();
    expect(screen.getByText(/1.2.4/i)).toBeInTheDocument();

    // Version count should be shown
    expect(screen.getByText(/2 versions/i)).toBeInTheDocument();

    // Last updated info should be shown
    expect(screen.getByText(/updated/i)).toBeInTheDocument();
  });

  it("renders empty state if no versions detected", () => {
    const versions: ISoftwareTitleVersion[] = [];

    render(
      <TitleVersionsTable
        router={mockRouter}
        data={versions}
        isLoading={false}
        teamIdForApi={42}
        isIPadOSOrIOSApp={false}
        countsUpdatedAt="2024-05-08T12:00:00Z"
      />
    );

    const cells = screen.queryAllByRole("cell");
    expect(cells).toHaveLength(0);

    // Version count should not be shown
    expect(screen.queryByText(/0 versions/i)).not.toBeInTheDocument();

    // Last updated info should be shown
    expect(screen.getByText(/updated/i)).toBeInTheDocument();

    // Empty state should be shown
    expect(screen.getByText(/no versions detected/i)).toBeInTheDocument();
  });
});
