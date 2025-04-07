import React from "react";
import { render, screen } from "@testing-library/react";
import { noop } from "lodash";
import DataTable from "./DataTable";

const DEFAULT_PAGE_SIZE = 20;

describe("DataTable - component", () => {
  it("renders a data table based on the columns and data passed in", () => {
    const columns = [
      {
        title: "Name",
        Header: "Name",
        accessor: "name",
        disableHidden: false,
      },
    ];

    const data = [{ name: "Test User", id: 1 }];

    render(
      <DataTable
        columns={columns}
        data={data}
        sortHeader="name"
        sortDirection="desc"
        isLoading={false}
        onSort={noop}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        resultsTitle="users"
        defaultPageSize={DEFAULT_PAGE_SIZE}
        disableMultiRowSelect={false}
      />
    );

    const nameHeader = screen.getByText("Name");
    const nameDataCell = screen.getByText("Test User");
    expect(nameHeader).toBeInTheDocument();
    expect(nameDataCell).toBeInTheDocument();
  });

  it("renders correctly given a sort header", () => {
    const columns = [
      {
        title: "Name",
        Header: "Name",
        accessor: "name",
        disableHidden: false,
      },
      {
        title: "Address",
        Header: "Address",
        accessor: "address",
        disableHidden: false,
      },
    ];

    // 'name' attribute is how we want to sort the data
    const { rerender } = render(
      <DataTable
        columns={columns}
        data={[
          { name: "foo user", address: "biz address" },
          { name: "bar user", address: "daz address" },
        ]}
        sortHeader="name"
        sortDirection="desc"
        isLoading={false}
        onSort={noop}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        resultsTitle="users"
        defaultPageSize={DEFAULT_PAGE_SIZE}
        disableMultiRowSelect={false}
      />
    );

    let dataCells = screen.getAllByRole("cell");
    let firstNameInTableCell = dataCells[0];
    let secondNameInTableCell = dataCells[2];
    expect(firstNameInTableCell).toHaveTextContent("foo user");
    expect(secondNameInTableCell).toHaveTextContent("bar user");

    // now want to sort on 'address' attribute
    rerender(
      <DataTable
        columns={columns}
        data={[
          { name: "foo user", address: "biz address" },
          { name: "bar user", address: "daz address" },
        ]}
        sortHeader="address"
        sortDirection="desc"
        isLoading={false}
        onSort={noop}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        resultsTitle="users"
        defaultPageSize={DEFAULT_PAGE_SIZE}
        disableMultiRowSelect={false}
      />
    );

    dataCells = screen.getAllByRole("cell");
    firstNameInTableCell = dataCells[1];
    secondNameInTableCell = dataCells[3];
    expect(firstNameInTableCell).toHaveTextContent("daz address");
    expect(secondNameInTableCell).toHaveTextContent("biz address");
  });

  it("renders correctly given a sortDirection", () => {
    const columns = [
      {
        title: "Name",
        Header: "Name",
        accessor: "name",
        disableHidden: false,
      },
    ];

    // testing 'desc' data
    const { rerender } = render(
      <DataTable
        columns={columns}
        data={[{ name: "foo user" }, { name: "bar user" }]}
        sortHeader="name"
        sortDirection="desc"
        isLoading={false}
        onSort={noop}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        resultsTitle="users"
        defaultPageSize={DEFAULT_PAGE_SIZE}
        disableMultiRowSelect={false}
      />
    );

    let dataCells = screen.getAllByRole("cell");
    let firstNameInTableCell = dataCells[0];
    let secondNameInTableCell = dataCells[1];
    expect(firstNameInTableCell).toHaveTextContent("foo user");
    expect(secondNameInTableCell).toHaveTextContent("bar user");

    // testing 'asc' data
    rerender(
      <DataTable
        columns={columns}
        data={[{ name: "foo user" }, { name: "bar user" }]}
        sortHeader="name"
        sortDirection="asc"
        isLoading={false}
        onSort={noop}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        resultsTitle="users"
        defaultPageSize={DEFAULT_PAGE_SIZE}
        disableMultiRowSelect={false}
      />
    );

    dataCells = screen.getAllByRole("cell");
    firstNameInTableCell = dataCells[0];
    secondNameInTableCell = dataCells[1];
    expect(firstNameInTableCell).toHaveTextContent("bar user");
    expect(secondNameInTableCell).toHaveTextContent("foo user");
  });

  it("does not render help text when no rows are present", () => {
    const columns = [
      {
        title: "Name",
        Header: "Name",
        accessor: "name",
        disableHidden: false,
      },
    ];

    const data: any = [];

    render(
      <DataTable
        columns={columns}
        data={data}
        sortHeader="name"
        sortDirection="desc"
        isLoading
        onSort={noop}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        resultsTitle="users"
        defaultPageSize={DEFAULT_PAGE_SIZE}
        disableMultiRowSelect={false}
        renderTableHelpText={() => <div>Help text</div>}
      />
    );

    const helpText = screen.queryByText("Help text");
    expect(helpText).toBeNull();
  });
  it("renders help text when rows are present", () => {
    const columns = [
      {
        title: "Name",
        Header: "Name",
        accessor: "name",
        disableHidden: false,
      },
    ];

    const data = [{ name: "Gabe" }];

    render(
      <DataTable
        columns={columns}
        data={data}
        sortHeader="name"
        sortDirection="desc"
        isLoading={false}
        onSort={noop}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        resultsTitle="users"
        defaultPageSize={DEFAULT_PAGE_SIZE}
        disableMultiRowSelect={false}
        renderTableHelpText={() => <div>Help text</div>}
      />
    );

    const helpText = screen.getByText("Help text");
    expect(helpText).toBeInTheDocument();
  });
});
