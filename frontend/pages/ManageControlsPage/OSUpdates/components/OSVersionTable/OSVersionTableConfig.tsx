import React from "react";

import { IDropdownOption } from "interfaces/dropdownOption";
import { IHostScript, ILastExecution } from "services/entities/scripts";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell";

interface IStatusCellProps {
  cell: {
    value: ILastExecution | null;
  };
}

interface IDropdownCellProps {
  cell: {
    value: IDropdownOption[];
  };
  row: {
    original: IHostScript;
  };
}

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

// eslint-disable-next-line import/prefer-default-export
export const generateTableHeaders = () => {
  return [
    {
      title: "OS type",
      Header: "OS type",
      disableSortBy: true,
      accessor: "platform",
    },
    {
      title: "Version",
      Header: "Version",
      disableSortBy: true,
      accessor: "version",
    },
    {
      title: "Hosts",
      accessor: "hosts_count",
      disableSortBy: false,
      Header: (cellProps: IHeaderProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
    },
  ];
};
