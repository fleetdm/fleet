import React from "react";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import StatusCell from "components/TableContainer/DataTable/StatusCell/StatusCell";
import TextCell from "components/TableContainer/DataTable/TextCell/TextCell";
import { IInvite } from "interfaces/invite";
import { IUser } from "interfaces/user";
import { IDropdownOption } from "interfaces/dropdownOption";
import { generateRole, generateTeam, greyCell } from "fleet/helpers";
import DropdownCell from "../../../components/TableContainer/DataTable/DropdownCell";

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

interface IDropdownCellProps extends IRowProps {
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
    | ((props: IDropdownCellProps) => JSX.Element);
  disableHidden?: boolean;
  disableSortBy?: boolean;
}

export interface IUserTableData {
  name: string;
  status: string;
  email: string;
  teams: string;
  role: string;
  actions: IDropdownOption[];
  id: number;
  type: string;
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
      Cell: (cellProps: ICellProps) => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      title: "Role",
      Header: "Role",
      accessor: "role",
      disableSortBy: true,
      Cell: (cellProps: ICellProps) => (
        <TextCell
          value={cellProps.cell.value}
          greyed={greyCell(cellProps.cell.value)}
        />
      ),
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
        <StatusCell value={cellProps.cell.value} />
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

  // Add Teams tab for premium tier only
  if (isPremiumTier) {
    tableHeaders.splice(2, 0, {
      title: "Teams",
      Header: "Teams",
      accessor: "teams",
      disableSortBy: true,
      Cell: (cellProps: ICellProps) => (
        <TextCell
          value={cellProps.cell.value}
          greyed={greyCell(cellProps.cell.value)}
        />
      ),
    });
  }

  return tableHeaders;
};

const generateStatus = (type: string, data: IUser | IInvite): string => {
  const { teams, global_role } = data;
  if (global_role === null && teams.length === 0) {
    return "No Access";
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
      name: user.name || "---",
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
    };
  });
};

const enhanceInviteData = (invites: IInvite[]): IUserTableData[] => {
  return invites.map((invite) => {
    return {
      name: invite.name || "---",
      status: generateStatus("invite", invite),
      email: invite.email,
      teams: generateTeam(invite.teams, invite.global_role),
      role: generateRole(invite.teams, invite.global_role),
      actions: generateActionDropdownOptions(false, true, invite.sso_enabled),
      id: invite.id,
      type: "invite",
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
