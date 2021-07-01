import React from "react";

import LinkCell from "components/TableContainer/DataTable/LinkCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import DropdownCell from "components/TableContainer/DataTable/DropdownCell";
// Changed to IQuery
import { IQuery } from "interfaces/query";
import { IDropdownOption } from "interfaces/dropdownOption";
import PATHS from "router/paths";

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

interface ICellProps {
  cell: {
    value: any;
  };
  row: {
    original: IQuery;
  };
}

interface IDataColumn {
  title: string;
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  accessor: string;
  Cell: (props: ICellProps) => JSX.Element;
  disableHidden?: boolean;
  disableSortBy?: boolean;
}

interface IQueryTableData extends IQuery {
  actions: IDropdownOption[];
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateTableHeaders = (
  actionSelectHandler: (value: string, query: IQuery) => void
): IDataColumn[] => {
  return [
    {
      title: "Name",
      Header: "Name",
      disableSortBy: true,
      accessor: "name",
      Cell: (cellProps) => (
        <LinkCell
          value={cellProps.cell.value}
          path={PATHS.TEAM_DETAILS_MEMBERS(cellProps.row.original.id)}
        />
      ),
    },
    // TODO: need to add this info to API
    {
      title: "Hosts",
      Header: "Hosts",
      disableSortBy: true,
      accessor: "host_count",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      title: "Members",
      Header: "Members",
      disableSortBy: true,
      accessor: "user_count",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      title: "Actions",
      Header: "",
      disableSortBy: true,
      accessor: "actions",
      Cell: (cellProps) => (
        <DropdownCell
          options={cellProps.cell.value}
          onChange={(value: string) =>
            actionSelectHandler(value, cellProps.row.original)
          }
          placeholder={"Actions"}
        />
      ),
    },
  ];
};

// NOTE: may need current user ID later for permission on actions.
const generateActionDropdownOptions = (): IDropdownOption[] => {
  return [
    {
      label: "Edit",
      disabled: false,
      value: "edit",
    },
    {
      label: "Delete",
      disabled: false,
      value: "delete",
    },
  ];
};

const enhanceQueryData = (queries: {
  [id: number]: IQuery;
}): IQueryTableData[] => {
  return Object.values(queries).map((query) => {
    return {
      // description: query.description,
      // name: query.name,
      // host_count: query.host_count,
      // user_count: query.user_count,
      // actions: generateActionDropdownOptions(),
      // id: query.id,
    };
  });
};

const generateDataSet = (queries: {
  [id: number]: IQuery;
}): IQueryTableData[] => {
  return [...enhanceQueryData(queries)];
};

export { generateTableHeaders, generateDataSet };
