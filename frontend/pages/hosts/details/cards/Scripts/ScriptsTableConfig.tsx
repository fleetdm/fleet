import React from "react";
import ReactTooltip from "react-tooltip";

import { COLORS } from "styles/var/colors";

import { IDropdownOption } from "interfaces/dropdownOption";
import { IHostScript, ILastExecution } from "services/entities/scripts";

import DropdownCell from "components/TableContainer/DataTable/DropdownCell";

import ScriptStatusCell from "./components/ScriptStatusCell";

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

const ScriptRunActionDropdownLabel = ({
  scriptId,
  disabled,
}: {
  scriptId: number;
  disabled: boolean;
}) => {
  const tipId = `run-script-${scriptId}`;
  return disabled ? (
    <>
      <span data-tip data-for={tipId}>
        Run
      </span>
      <ReactTooltip
        place="bottom"
        type="dark"
        effect="solid"
        id={tipId}
        backgroundColor={COLORS["tooltip-bg"]}
        delayHide={100}
        delayUpdate={500}
      >
        You can only run the script when the host is online.
      </ReactTooltip>
    </>
  ) : (
    <>Run</>
  );
};

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
        return <ScriptStatusCell lastExecution={value} />;
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
  { script_id, last_execution }: IHostScript,
  isHostOnline: boolean
): IDropdownOption[] => {
  return [
    {
      label: "Show details",
      disabled: last_execution === null,
      value: "showDetails",
    },
    {
      label: (
        <ScriptRunActionDropdownLabel
          scriptId={script_id}
          disabled={!isHostOnline}
        />
      ),
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
