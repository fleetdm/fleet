import React from "react";

import ActionsDropdown from "components/ActionsDropdown";
import ApiEndpointCountTag from "components/ApiEndpointCountTag";
import ApiUserTag from "components/ApiUserTag";
import TextCell from "components/TableContainer/DataTable/TextCell/TextCell";
import TooltipTruncatedTextCell from "components/TableContainer/DataTable/TooltipTruncatedTextCell";
import TooltipWrapper from "components/TooltipWrapper";
import { IDropdownOption } from "interfaces/dropdownOption";
import { ITeam } from "interfaces/team";
import { IUser, UserRole } from "interfaces/user";
import permissions from "utilities/permissions";
import stringUtils from "utilities/strings";

const baseClass = "team-users";

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

interface IRowProps {
  row: {
    original: IUser;
  };
}

interface ICellProps extends IRowProps {
  cell: {
    value: string | number | boolean;
  };
}

interface IPermissionsCellProps {
  cell: {
    value: string;
  };
  row: {
    original: ITeamUsersTableData;
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
  id?: string;
  Cell:
    | ((props: ICellProps) => JSX.Element)
    | ((props: IPermissionsCellProps) => JSX.Element)
    | ((props: IActionsDropdownProps) => JSX.Element);
  disableHidden?: boolean;
  disableSortBy?: boolean;
  sortType?: string;
}

export interface ITeamUsersTableData {
  name: string;
  email: string;
  role: UserRole;
  teams: ITeam[];
  actions: IDropdownOption[];
  id: number;
  api_only?: boolean;
  apiEndpointCount: number;
}

const renderRole = (cellProps: IPermissionsCellProps) => {
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
            ability to run any live report against all hosts.
          </>
        }
      >
        {cellProps.cell.value}
      </TooltipWrapper>
    );
  }
  return <TextCell value={cellProps.cell.value} className="permissions-text" />;
};

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateColumnConfigs = (
  actionSelectHandler: (value: string, user: IUser) => void,
  currentUser: IUser | null
): IDataColumn[] => {
  return [
    {
      title: "Name",
      Header: "Name",
      disableSortBy: true,
      sortType: "caseInsensitive",
      accessor: "name",
      Cell: (cellProps: ICellProps) => {
        const apiOnlyUser =
          "api_only" in cellProps.row.original
            ? cellProps.row.original.api_only
            : false;

        return (
          <TooltipTruncatedTextCell
            value={cellProps.cell.value}
            suffix={apiOnlyUser && <ApiUserTag />}
          />
        );
      },
    },
    {
      title: "Permissions",
      Header: "Permissions",
      disableSortBy: true,
      accessor: "role",
      id: "permissions",
      Cell: (cellProps: IPermissionsCellProps) => {
        const { apiEndpointCount } = cellProps.row.original;

        return (
          <div className={`${baseClass}__permissions-content`}>
            {renderRole(cellProps)}
            {apiEndpointCount > 0 && (
              <ApiEndpointCountTag count={apiEndpointCount} />
            )}
          </div>
        );
      },
    },
    {
      title: "Email",
      Header: "Email",
      disableSortBy: true,
      accessor: "email",
      Cell: (cellProps: ICellProps) => (
        <TextCell className="w400" value={cellProps.cell.value} />
      ),
    },
    {
      title: "Actions",
      Header: "",
      disableSortBy: true,
      accessor: "actions",
      Cell: (cellProps: IActionsDropdownProps) => {
        const rowUser = cellProps.row.original;
        // A team admin can only manage users that are members of fleets they
        // administer. If the user also belongs to other fleets the current user
        // is not an admin of, the actions are disabled (mirrors the backend
        // authorization rule).
        const canManageUser = permissions.isAdminForAllUserTeams(
          currentUser,
          rowUser
        );

        const dropdown = (
          <ActionsDropdown
            options={cellProps.cell.value}
            onChange={(value: string) => actionSelectHandler(value, rowUser)}
            placeholder="Actions"
            variant="secondary"
            disabled={!canManageUser}
          />
        );

        if (canManageUser) {
          return dropdown;
        }

        return (
          <TooltipWrapper
            position="top"
            showArrow
            underline={false}
            tipContent={
              <>
                This user can&apos;t be managed because they&apos;re a member of
                other fleets you&apos;re not an admin of.
              </>
            }
          >
            {dropdown}
          </TooltipWrapper>
        );
      },
    },
  ];
};

const generateActionDropdownOptions = (): IDropdownOption[] => {
  return [
    {
      label: "Edit",
      disabled: false,
      value: "edit",
    },
    {
      label: "Remove",
      disabled: false,
      value: "remove",
    },
  ];
};
const generateRole = (teamId: number, teams: ITeam[]): UserRole => {
  const role = teams.find((team) => teamId === team.id)?.role ?? "Unassigned";
  return stringUtils.capitalizeRole(role);
};

const enhanceUsersData = (
  teamId: number,
  users: IUser[]
): ITeamUsersTableData[] => {
  return Object.values(users).map((user) => {
    return {
      name: user.name,
      email: user.email,
      role: generateRole(teamId, user.teams),
      teams: user.teams,
      sso_enabled: user.sso_enabled,
      mfa_enabled: user.mfa_enabled,
      global_role: user.global_role,
      actions: generateActionDropdownOptions(),
      id: user.id,
      api_only: user.api_only,
      apiEndpointCount: user.api_endpoints?.length ?? 0,
    };
  });
};

const generateDataSet = (
  teamId: number,
  users: IUser[]
): ITeamUsersTableData[] => {
  return [...enhanceUsersData(teamId, users)];
};

export { generateColumnConfigs, generateDataSet };
