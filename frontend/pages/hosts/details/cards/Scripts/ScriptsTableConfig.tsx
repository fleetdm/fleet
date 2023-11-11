import React from "react";
import ReactTooltip from "react-tooltip";

import { COLORS } from "styles/var/colors";

import { IDropdownOption } from "interfaces/dropdownOption";
import { IHostScript, ILastExecution } from "services/entities/scripts";
import { IUser } from "interfaces/user";

import DropdownCell from "components/TableContainer/DataTable/DropdownCell";
import { IHost } from "interfaces/host";
import {
  isGlobalAdmin,
  isTeamMaintainer,
  isTeamAdmin,
  isGlobalMaintainer,
  isGlobalObserver,
  isTeamObserver,
} from "utilities/permissions/permissions";

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
export const generateTableColumnConfigs = (
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

const generateActionDropdownOptions = (
  currentUser: IUser | null,
  host: IHost,
  { script_id, last_execution }: IHostScript
): IDropdownOption[] => {
  // Run action available to {globalteam} maintainer, {global|team} admin, {global|team} observers+,
  // {global|team} observers

  const [hostTeamId, isHostOnline] = [host.team_id, host.status === "online"];

  const hasRunPermission =
    !!currentUser &&
    (isGlobalAdmin(currentUser) ||
      isTeamAdmin(currentUser, hostTeamId) ||
      isGlobalMaintainer(currentUser) ||
      isTeamMaintainer(currentUser, hostTeamId) ||
      // TODO - refactor all permissions to be clear and granular
      // each of these (confusingly) cover both observer and observer+
      isGlobalObserver(currentUser) ||
      isTeamObserver(currentUser, hostTeamId));
  const options: IDropdownOption[] = [
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
  return hasRunPermission ? options : options.slice(0, 1);
};

export const generateDataSet = (
  currentUser: IUser | null,
  host: IHost,
  scripts: IHostScript[]
) => {
  return scripts.map((script) => {
    return {
      ...script,
      actions: generateActionDropdownOptions(currentUser, host, script),
    };
  });
};
