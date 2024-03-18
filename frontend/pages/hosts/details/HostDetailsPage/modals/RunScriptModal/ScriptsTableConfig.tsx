import React from "react";
import ReactTooltip from "react-tooltip";
import { noop } from "lodash";

import { COLORS } from "styles/var/colors";

import { IDropdownOption } from "interfaces/dropdownOption";
import { IHostScript, ILastExecution } from "interfaces/script";
import { IUser } from "interfaces/user";

import Icon from "components/Icon";
import DropdownCell from "components/TableContainer/DataTable/DropdownCell";
import {
  isGlobalAdmin,
  isTeamMaintainer,
  isTeamAdmin,
  isGlobalMaintainer,
  isGlobalObserver,
  isTeamObserver,
} from "utilities/permissions/permissions";
import TooltipWrapper from "components/TooltipWrapper";

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
        Script is already running.
      </ReactTooltip>
    </>
  ) : (
    <>Run</>
  );
};

const generateActionDropdownOptions = (
  currentUser: IUser | null,
  teamId: number | null,
  { script_id, last_execution }: IHostScript
): IDropdownOption[] => {
  const hasRunPermission =
    !!currentUser &&
    (isGlobalAdmin(currentUser) ||
      isTeamAdmin(currentUser, teamId) ||
      isGlobalMaintainer(currentUser) ||
      isTeamMaintainer(currentUser, teamId) ||
      // TODO - refactor all permissions to be clear and granular
      // each of these (confusingly) cover both observer and observer+
      isGlobalObserver(currentUser) ||
      isTeamObserver(currentUser, teamId));
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
          disabled={last_execution?.status === "pending"}
        />
      ),
      disabled: last_execution?.status === "pending",
      value: "run",
    },
  ];
  return hasRunPermission ? options : options.slice(0, 1);
};

// eslint-disable-next-line import/prefer-default-export
export const generateTableColumnConfigs = (
  currentUser: IUser | null,
  hostTeamId: number | null,
  scriptsDisabled: boolean,
  onSelectAction: (value: string, script: IHostScript) => void
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
      Cell: (cellProps: IDropdownCellProps) => {
        if (scriptsDisabled) {
          // create a basic span that doesn't use the dropdown component (which relies on react-select
          // and makes it difficult for us to style the disabled tooltip underline on the placeholder text.
          return (
            <span className="run-script-action--disabled">
              <TooltipWrapper
                position="top"
                tipContent={
                  <div>
                    Running scripts is disabled in organization settings
                  </div>
                }
              >
                Actions
              </TooltipWrapper>
              <Icon name="chevron-down" color="ui-fleet-black-50" />
            </span>
          );
        }

        const opts = generateActionDropdownOptions(
          currentUser,
          hostTeamId,
          cellProps.row.original
        );
        return (
          <DropdownCell
            options={opts}
            onChange={(value: string) =>
              onSelectAction(value, cellProps.row.original)
            }
            placeholder="Actions"
            disabled={scriptsDisabled}
          />
        );
      },
    },
  ];
};
