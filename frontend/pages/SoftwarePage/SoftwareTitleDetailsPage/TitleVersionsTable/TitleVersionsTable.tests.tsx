import React from "react";
import { screen, render, within } from "@testing-library/react";
import { createMockRouter, renderWithSetup } from "test/test-utils";
import { ISoftwareTitleVersion } from "interfaces/software";
import TitleVersionsTable from "./TitleVersionsTable";

const mockRouter = createMockRouter();

describe("TitleVersionsTable", () => {
  // Deliberately ordered so that the input order, host count order, and
  // lexical and numeric version orderings are all different. 1.2.10 vs 1.2.9
  // catches a lexical version sort, and 10 vs 9 vs 2 a lexical host count one.
  const unsortedVersions = [
    { id: 1, version: "1.2.9", vulnerabilities: [], hosts_count: 2 },
    { id: 2, version: "1.2.2", vulnerabilities: [], hosts_count: 10 },
    { id: 3, version: "1.2.10", vulnerabilities: [], hosts_count: 9 },
  ];

  const renderTable = (data = unsortedVersions) =>
    renderWithSetup(
      <TitleVersionsTable
        router={mockRouter}
        data={data}
        isLoading={false}
        teamIdForApi={42}
        isIPadOSOrIOSApp={false}
        countsUpdatedAt="2024-05-08T12:00:00Z"
      />
    );

  const renderedVersions = () =>
    screen
      .getAllByRole("row")
      .slice(1) // the first row is the header
      .map((row) => within(row).getAllByRole("cell")[0].textContent);

  it("sorts by host count by default", () => {
    renderTable();

    expect(renderedVersions()).toEqual(["1.2.2", "1.2.10", "1.2.9"]);
  });

  it("sorts by host count when the Hosts header is clicked", async () => {
    const { user } = renderTable();
    const hostsHeader = screen.getByRole("button", { name: "Hosts" });

    // Hosts is the default sorted column, so the first click reverses it.
    await user.click(hostsHeader);
    expect(renderedVersions()).toEqual(["1.2.9", "1.2.10", "1.2.2"]);

    await user.click(hostsHeader);
    expect(renderedVersions()).toEqual(["1.2.2", "1.2.10", "1.2.9"]);

    // Coming back from another sorted column re-sorts on host count, rather
    // than flipping the direction of whichever column was already sorted.
    await user.click(screen.getByRole("button", { name: "Version" }));
    await user.click(hostsHeader);
    expect(renderedVersions()).toEqual(["1.2.9", "1.2.10", "1.2.2"]);
  });

  it("sorts by version number when the Version header is clicked", async () => {
    const { user } = renderTable();
    const versionHeader = screen.getByRole("button", { name: "Version" });

    await user.click(versionHeader);
    expect(renderedVersions()).toEqual(["1.2.2", "1.2.9", "1.2.10"]);

    await user.click(versionHeader);
    expect(renderedVersions()).toEqual(["1.2.10", "1.2.9", "1.2.2"]);
  });

  it("treats trailing zero segments as the same version", async () => {
    // Distinguishes the "version" sortType from react-table's default sort,
    // which would order 14.3 ahead of 14.3.0 instead of leaving them tied.
    const { user } = renderTable([
      { id: 1, version: "14.3.0", vulnerabilities: [], hosts_count: 1 },
      { id: 2, version: "14.3", vulnerabilities: [], hosts_count: 2 },
    ]);

    await user.click(screen.getByRole("button", { name: "Version" }));

    expect(renderedVersions()).toEqual(["14.3.0", "14.3"]);
  });

  it("renders version names as links and footer info", () => {
    const versions = [
      { id: 10, version: "1.2.3", vulnerabilities: [] },
      { id: 11, version: "1.2.4", vulnerabilities: [] },
      { id: 12, version: "1.2.5", vulnerabilities: [] },
      { id: 13, version: "1.2.6", vulnerabilities: [] },
      { id: 14, version: "1.2.7", vulnerabilities: [] },
      { id: 15, version: "1.2.8", vulnerabilities: [] },
      { id: 16, version: "1.2.9", vulnerabilities: [] },
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
    // Version, Vulnerabilities, Hosts, View all hosts for each row
    expect(cells).toHaveLength(4 * 7);
    expect(screen.getByText(/1.2.3/i)).toBeInTheDocument();
    expect(screen.getByText(/1.2.9/i)).toBeInTheDocument(); // make sure

    // Version count should be shown
    expect(screen.getByText(/7 versions/i)).toBeInTheDocument();

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
