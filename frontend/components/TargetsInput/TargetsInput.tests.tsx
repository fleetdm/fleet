import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { noop } from "lodash";
import React from "react";

import createMockHost from "__mocks__/hostMock";
import { IHost } from "interfaces/host";

import TargetsInput from "./TargetsInput";
import {
  generateTableHeaders,
  ITargestInputHostTableConfig,
} from "./TargetsInputHostsTableConfig";

describe("TargetsInput", () => {
  it("renders the search table based on the custom configuration passed in", () => {
    const testHosts: IHost[] = [
      createMockHost({
        display_name: "testHost",
        public_ip: "123.456.789.0",
        computer_name: "testName",
      }),
    ];

    const testTableConfig: ITargestInputHostTableConfig[] = [
      {
        Header: "Name",
        accessor: "display_name",
      },
      {
        Header: "IP Address",
        accessor: "public_ip",
      },
    ];

    render(
      <TargetsInput
        searchText="test"
        searchResults={testHosts}
        isTargetsLoading={false}
        hasFetchError={false}
        targetedHosts={[]}
        searchResultsTableConfig={testTableConfig}
        selectedHostsTableConifg={[]}
        setSearchText={noop}
        handleRowSelect={noop}
      />
    );

    expect(screen.getByText("Name")).toBeInTheDocument();
    expect(screen.getByText("IP Address")).toBeInTheDocument();
    expect(screen.getByText("testHost")).toBeInTheDocument();
    expect(screen.getByText("123.456.789.0")).toBeInTheDocument();
    expect(screen.queryByText("testName")).not.toBeInTheDocument();
  });

  it("renders the results table based on the custom configuration passed in", () => {
    const testHosts: IHost[] = [
      createMockHost({
        display_name: "testHost",
        public_ip: "123.456.789.0",
        computer_name: "testName",
      }),
    ];

    const testTableConfig: ITargestInputHostTableConfig[] = [
      {
        Header: "Name",
        accessor: "display_name",
      },
      {
        Header: "IP Address",
        accessor: "public_ip",
      },
    ];

    render(
      <TargetsInput
        searchText=""
        searchResults={[]}
        isTargetsLoading={false}
        hasFetchError={false}
        targetedHosts={testHosts}
        searchResultsTableConfig={[]}
        selectedHostsTableConifg={testTableConfig}
        setSearchText={noop}
        handleRowSelect={noop}
      />
    );

    expect(screen.getByText("Name")).toBeInTheDocument();
    expect(screen.getByText("IP Address")).toBeInTheDocument();
    expect(screen.getByText("testHost")).toBeInTheDocument();
    expect(screen.getByText("123.456.789.0")).toBeInTheDocument();
    expect(screen.queryByText("testName")).not.toBeInTheDocument();
  });

  it("sorts the OS column by the name and version it shows", async () => {
    const testHosts: IHost[] = [
      createMockHost({ id: 1, os_name: "Ubuntu", os_version: "22.04" }),
      createMockHost({ id: 2, os_name: "Fedora Linux", os_version: "40" }),
    ];

    render(
      <TargetsInput
        searchText=""
        searchResults={[]}
        isTargetsLoading={false}
        hasFetchError={false}
        targetedHosts={testHosts}
        searchResultsTableConfig={[]}
        selectedHostsTableConifg={generateTableHeaders()}
        setSearchText={noop}
        handleRowSelect={noop}
      />
    );

    await userEvent.click(screen.getByRole("button", { name: "OS" }));

    const osCells = screen.getAllByText(/^(Ubuntu 22\.04|Fedora Linux 40)$/);
    expect(osCells.map((cell) => cell.textContent)).toEqual([
      "Fedora Linux 40",
      "Ubuntu 22.04",
    ]);
  });
});
