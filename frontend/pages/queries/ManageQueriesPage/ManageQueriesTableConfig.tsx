import React from "react";

import moment from "moment";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell/TextCell";
import { IQuery } from "interfaces/query";

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

interface IQueryTableData {
  name: string;
  description: string;
  observer_can_run: string | boolean;
  author_name: string;
  last_modified: string;
  // status: string;
  // email: string;
  // teams: string;
  // roles: string;
  // actions: IDropdownOption[];
  // id: number;
  // type: string;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateTableHeaders = (isOnlyObserver = false): IDataColumn[] => {
  const tableHeaders: IDataColumn[] = [
    {
      title: "Name",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "name",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      title: "Description",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "description",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      title: "Author",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "author_name",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      title: "Last modified",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "last_modified",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },

    // {
    //   title: "Actions",
    //   Header: "Actions",
    //   disableSortBy: true,
    //   accessor: "actions",
    //   Cell: (cellProps) => (
    //     <DropdownCell
    //       options={cellProps.cell.value}
    //       onChange={(value: string) =>
    //         actionSelectHandler(value, cellProps.row.original)
    //       }
    //       placeholder={"Actions"}
    //     />
    //   ),
    // },
  ];

  // Add Teams tab for basic tier only
  if (!isOnlyObserver) {
    tableHeaders.splice(3, 0, {
      title: "Observers can run",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "observer_can_run",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    });
  }

  return tableHeaders;
};

const generateTableData = (queries: IQuery[]): IQueryTableData[] => {
  return queries.map((query) => {
    return {
      name: query.name || "---",
      description: query.description || "--",
      observer_can_run: query.observer_can_run,
      author_name: query.author_name,
      last_modified: moment(query.updated_at).format("MM/DD/YY"),

      // status: generateStatus("user", user),
      // email: user.email,
      // teams: generateTeam(user.teams, user.global_role),
      // roles: generateRole(user.teams, user.global_role),
      // actions: generateActionDropdownOptions(
      //   user.id === currentUserId,
      //   false,
      //   user.sso_enabled
      // ),
      // id: user.id,
      // type: "user",
    };
  });
};

export { generateTableHeaders, generateTableData };
