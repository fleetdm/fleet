import React from "react";

import { IDropdownOption } from "interfaces/dropdownOption";
import { IScriptExecutionStatus } from "services/entities/scripts";

import ScriptStatusCell from "./components/ScriptStatusCell";

interface IStatusCellProps {
  cell: {
    value: IScriptExecutionStatus | null;
  };
}

interface IDropdownCellProps {
  cell: {
    value: IDropdownOption[];
  };
}

const DEFAULT_TABLE_HEADERS = [
  {
    title: "Name",
    Header: "Name",
    disableSortBy: true,
    accessor: "name",
  },
  {
    title: "Status",
    Header: "Status",
    disableSortBy: true,
    accessor: "last_execution",
    Cell: ({ cell: { value } }: IStatusCellProps) => {
      return <ScriptStatusCell status={value} />;
    },
  },
  {
    title: "Actions",
    Header: "",
    disableSortBy: true,
    accessor: "actions",
  },
];

// eslint-disable-next-line import/prefer-default-export
export const generateTableHeaders = () => {
  return DEFAULT_TABLE_HEADERS;
};
