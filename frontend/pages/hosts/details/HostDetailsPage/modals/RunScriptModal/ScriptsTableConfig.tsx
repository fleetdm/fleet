import React from "react";

import { IDropdownOption } from "interfaces/dropdownOption";
import { IHostScript, ILastExecution } from "interfaces/script";
import { IUser } from "interfaces/user";

import Icon from "components/Icon";
import ActionsDropdown from "components/ActionsDropdown";
import {
  isGlobalAdmin,
  isTeamMaintainer,
  isTeamAdmin,
  isGlobalMaintainer,
  isGlobalObserver,
  isTeamObserver,
} from "utilities/permissions/permissions";
import Button from "components/buttons/Button";
import TooltipWrapper from "components/TooltipWrapper";

import ScriptStatusCell from "./components/ScriptStatusCell";

interface IRowProps {
  row: {
    original: IHostScript;
  };
}
interface ICellProps extends IRowProps {
  cell: {
    value: string;
  };
}

interface IStatusCellProps {
  cell: {
    value: ILastExecution | null;
  };
}

interface IActionsDropdownProps {
  cell: {
    value: IDropdownOption[];
  };
  row: {
    original: IHostScript;
  };
}

export const generateActionDropdownOptions = (
  currentUser: IUser | null,
  teamId: number | null,
  { last_execution }: IHostScript
): IDropdownOption[] => {
  const isPending = last_execution?.status === "pending";
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
      label: "Show run details",
      disabled: last_execution === null,
      value: "showRunDetails",
    },
  ];
  hasRunPermission &&
    options.unshift({
      label: "Run",
      disabled: isPending,
      value: "run",
      tooltipContent: isPending ? "Script is already running." : undefined,
    });
  return options;
};

// eslint-disable-next-line import/prefer-default-export
export const generateTableColumnConfigs = (
  currentUser: IUser | null,
  hostTeamId: number | null,
  scriptsDisabled: boolean,
  onClickViewScript: (scriptId: number, scriptDetails: IHostScript) => void,
  onSelectAction: (value: string, script: IHostScript) => void
) => {
  return [
    {
      title: "Name",
      Header: "Name",
      disableSortBy: true,
      accessor: "name",
      Cell: (cellProps: ICellProps) => {
        const { name, script_id } = cellProps.row.original;

        const onClickScriptName = (e: React.MouseEvent) => {
          // Allows for button to be clickable in a clickable row
          e.stopPropagation();
          onClickViewScript(script_id, cellProps.row.original);
        };

        return (
          <Button
            className="script-info"
            onClick={onClickScriptName}
            variant="text-icon"
          >
            <span className={`script-info-text`}>{name}</span>
          </Button>
        );
      },
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
      Cell: (cellProps: IActionsDropdownProps) => {
        if (scriptsDisabled) {
          // create a basic span that doesn't use the dropdown component (which relies on react-select
          // and makes it difficult for us to style the disabled tooltip underline on the placeholder text.
          return (
            <span className="run-script-action--disabled">
              <TooltipWrapper
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
          <ActionsDropdown
            options={opts}
            onChange={(value: string) =>
              onSelectAction(value, cellProps.row.original)
            }
            placeholder="Actions"
            disabled={scriptsDisabled}
            menuAlign="right"
          />
        );
      },
    },
  ];
};
