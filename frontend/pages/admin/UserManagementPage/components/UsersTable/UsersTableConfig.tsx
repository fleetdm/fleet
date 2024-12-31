import React from "react";

import ReactTooltip from "react-tooltip";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import StatusIndicator from "components/StatusIndicator";
import TextCell from "components/TableContainer/DataTable/TextCell/TextCell";
import CustomLink from "components/CustomLink";
import TooltipWrapper from "components/TooltipWrapper";
import { IInvite } from "interfaces/invite";
import { IUser, UserRole } from "interfaces/user";
import { IDropdownOption } from "interfaces/dropdownOption";
import { generateRole, generateTeam, greyCell } from "utilities/helpers";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import { COLORS } from "styles/var/colors";
import ActionsDropdown from "../../../../../components/ActionsDropdown";

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

interface IRowProps {
  row: {
    original: IUser | IInvite;
  };
}

interface ICellProps extends IRowProps {
  cell: {
    value: string;
  };
}

interface IActionsDropdownProps extends IRowProps {
  cell: {
    value: IDropdownOption[];
  };
}

interface IDataColumn {
  title: string;
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  accessor: string;
  Cell:
    | ((props: ICellProps) => JSX.Element)
    | ((props: IActionsDropdownProps) => JSX.Element);
  disableHidden?: boolean;
  disableSortBy?: boolean;
}

export interface IUserTableData {
  name: string;
  status: string;
  email: string;
  teams: string;
  role: UserRole;
  actions: IDropdownOption[];
  id: number;
  type: string;
  api_only: boolean;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateTableHeaders = (
  actionSelectHandler: (value: string, user: IUser | IInvite) => void,
  isPremiumTier: boolean | undefined
): IDataColumn[] => {
  const tableHeaders: IDataColumn[] = [
    {
      title: "Name",
      Header: "Name",
      disableSortBy: true,
      accessor: "name",
      Cell: (cellProps: ICellProps) => {
        const formatter = (val: string) => {
          const apiOnlyUser =
            "api_only" in cellProps.row.original
              ? cellProps.row.original.api_only
              : false;

          return (
            <>
              {val}
              {apiOnlyUser && (
                <>
                  <span
                    className="user-management__api-only-user"
                    data-tip
                    data-for={`api-only-tooltip-${cellProps.row.original.id}`}
                  >
                    API
                  </span>
                  <ReactTooltip
                    className="api-only-tooltip"
                    place="top"
                    type="dark"
                    effect="solid"
                    id={`api-only-tooltip-${cellProps.row.original.id}`}
                    backgroundColor={COLORS["tooltip-bg"]}
                    clickable
                    delayHide={200} // need delay set to hover using clickable
                  >
                    <>
                      This user was created using fleetctl and
                      <br /> only has API access.{" "}
                      <CustomLink
                        text="Learn more"
                        newTab
                        url="https://fleetdm.com/docs/using-fleet/fleetctl-cli#using-fleetctl-with-an-api-only-user"
                        iconColor="core-fleet-white"
                      />
                    </>
                  </ReactTooltip>
                </>
              )}
            </>
          );
        };

        return <TextCell value={cellProps.cell.value} formatter={formatter} />;
      },
    },
    {
      title: "Role",
      Header: "Role",
      accessor: "role",
      disableSortBy: true,
      Cell: (cellProps: ICellProps) => {
        if (cellProps.cell.value === "GitOps") {
          return (
            <TooltipWrapper
              tipContent={
                <>
                  The GitOps role is only available on the command-line
                  <br />
                  when creating an API-only user. This user has no
                  <br />
                  access to the UI.
                </>
              }
            >
              GitOps
            </TooltipWrapper>
          );
        }
        if (cellProps.cell.value === "Observer+") {
          return (
            <TooltipWrapper
              tipContent={
                <>
                  Users with the Observer+ role have access to all of
                  <br />
                  the same functions as an Observer, with the added
                  <br />
                  ability to run any live query against all hosts.
                </>
              }
            >
              {cellProps.cell.value}
            </TooltipWrapper>
          );
        }
        const greyAndItalic = greyCell(cellProps.cell.value);
        return (
          <TextCell
            value={cellProps.cell.value}
            grey={greyAndItalic}
            italic={greyAndItalic}
          />
        );
      },
    },
    {
      title: "Status",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "status",
      Cell: (cellProps: ICellProps) => (
        <StatusIndicator value={cellProps.cell.value} />
      ),
    },
    {
      title: "Email",
      Header: "Email",
      disableSortBy: true,
      accessor: "email",
      Cell: (cellProps: ICellProps) => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      title: "Actions",
      Header: "",
      disableSortBy: true,
      accessor: "actions",
      Cell: (cellProps: IActionsDropdownProps) => (
        <ActionsDropdown
          options={cellProps.cell.value}
          onChange={(value: string) =>
            actionSelectHandler(value, cellProps.row.original)
          }
          placeholder="Actions"
        />
      ),
    },
  ];

  // Add Teams column for premium tier
  if (isPremiumTier) {
    tableHeaders.splice(2, 0, {
      title: "Teams",
      Header: "Teams",
      accessor: "teams",
      disableSortBy: true,
      Cell: (cellProps: ICellProps) => (
        <TextCell value={cellProps.cell.value} />
      ),
    });
  }

  return tableHeaders;
};

const generateStatus = (type: string, data: IUser | IInvite): string => {
  const { teams, global_role } = data;
  if (global_role === null && teams.length === 0) {
    return "No access";
  }

  return type === "invite" ? "Invite pending" : "Active";
};

const generateActionDropdownOptions = (
  isCurrentUser: boolean,
  isInvitePending: boolean,
  isSsoEnabled: boolean
): IDropdownOption[] => {
  let dropdownOptions = [
    {
      label: "Edit",
      disabled: false,
      value: isCurrentUser ? "editMyAccount" : "edit",
    },
    {
      label: "Require password reset",
      disabled: isInvitePending,
      value: "passwordReset",
    },
    {
      label: "Reset sessions",
      disabled: isInvitePending,
      value: "resetSessions",
    },
    {
      label: "Delete",
      disabled: isCurrentUser,
      value: "delete",
    },
  ];

  if (isCurrentUser) {
    // remove "Reset sessions" from dropdownOptions
    dropdownOptions = dropdownOptions.filter(
      (option) => option.label !== "Reset sessions"
    );
  }

  if (isSsoEnabled) {
    // remove "Require password reset" from dropdownOptions
    dropdownOptions = dropdownOptions.filter(
      (option) => option.label !== "Require password reset"
    );
  }
  return dropdownOptions;
};

const enhanceUserData = (
  users: IUser[],
  currentUserId: number
): IUserTableData[] => {
  return users.map((user) => {
    return {
      name: user.name || DEFAULT_EMPTY_CELL_VALUE,
      status: generateStatus("user", user),
      email: user.email,
      teams: generateTeam(user.teams, user.global_role),
      role: generateRole(user.teams, user.global_role),
      actions: generateActionDropdownOptions(
        user.id === currentUserId,
        false,
        user.sso_enabled
      ),
      id: user.id,
      type: "user",
      api_only: user.api_only,
    };
  });
};

const enhanceInviteData = (invites: IInvite[]): IUserTableData[] => {
  return invites.map((invite) => {
    return {
      name: invite.name || DEFAULT_EMPTY_CELL_VALUE,
      status: generateStatus("invite", invite),
      email: invite.email,
      teams: generateTeam(invite.teams, invite.global_role),
      role: generateRole(invite.teams, invite.global_role),
      actions: generateActionDropdownOptions(false, true, invite.sso_enabled),
      id: invite.id,
      type: "invite",
      api_only: false, // api only users are created through fleetctl and not invites
    };
  });
};

const combineDataSets = (
  users: IUser[],
  invites: IInvite[],
  currentUserId: number
): IUserTableData[] => {
  return [
    ...enhanceUserData(users, currentUserId),
    ...enhanceInviteData(invites),
  ];
};

export { generateTableHeaders, combineDataSets };
