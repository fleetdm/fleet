import React from "react";

import { IDropdownOption } from "interfaces/dropdownOption";
import { IHostScript, IScriptExecutionStatus } from "services/entities/scripts";

import DropdownCell from "components/TableContainer/DataTable/DropdownCell";

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
  row: {
    original: IHostScript;
  };
}

// eslint-disable-next-line import/prefer-default-export
export const generateTableHeaders = (
  actionSelectHandler: (value: string, script: IHostScript) => void
) => {
  return [
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
      Cell: (cellProps: IDropdownCellProps) => (
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
const generateActionDropdownOptions = (
  script: IHostScript,
  isHostOnline: boolean
): IDropdownOption[] => {
  return [
    {
      label: "Show details",
      disabled: script.last_execution === null,
      value: "showDetails",
    },
    {
      label: "Run",
      disabled: !isHostOnline,
      value: "run",
    },
  ];
};

export const generateDataSet = (
  scripts: IHostScript[],
  isHostOnline: boolean
) => {
  return scripts.map((script) => {
    return {
      ...script,
      actions: generateActionDropdownOptions(script, isHostOnline),
    };
  });
};
