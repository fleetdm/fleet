import React from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { noop } from "lodash";
// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import DataTable from "./DataTable";

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
        isLoading={false}
        onSort={noop}
      />
    );

    const nameHeader = screen.getByText("Name");
    const nameDataCell = screen.getByText("Test User");
    expect(nameHeader).toBeInTheDocument();
    expect(nameDataCell).toBeInTheDocument();
  });

  it("renders correctly given the sort header and direction", () => {
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

    const data = [{ name: "Foo User" }, { name: "Bar User" }];

    render(
      <DataTable
        columns={columns}
        data={data}
        sortHeader={"name"}
        sortDirection={"desc"}
        isLoading={false}
        onSort={noop}
      />
    );

    const nameHeader = screen.queryByRole("cell");
    expect(nameHeader).toBeInTheDocument();
  });
});
