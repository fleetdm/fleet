import React from "react";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import StatusIndicator from "components/StatusIndicator";
import TextCell from "components/TableContainer/DataTable/TextCell/TextCell";
import TooltipTruncatedTextCell from "components/TableContainer/DataTable/TooltipTruncatedTextCell";
import TooltipWrapper from "components/TooltipWrapper";
import Tag from "components/Tag";
import { IInvite } from "interfaces/invite";
import { IUser, UserRole, UserStatus } from "interfaces/user";
import { IDropdownOption } from "interfaces/dropdownOption";
import {
  generateRole,
  generateRoleGroups,
  generateTeam,
  generateTeamNames,
  greyCell,
  ROLE_VARIOUS,
  ROLE_GLOBAL,
  tooltipTextWithLineBreaks,
} from "utilities/helpers";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import ActionsDropdown from "../../../../../components/ActionsDropdown";

const baseClass = "users-table";

const renderApiUserIndicator = () => {
  return (
    <Tag tooltip="This user only has API access." size="xsmall">
      API
    </Tag>
  );
};

const renderApiEndpointCount = (count: number) => (
  <Tag size="xsmall">{`${count} API endpoint${count === 1 ? "" : "s"}`}</Tag>
);

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

interface IRowProps {
  row: {
    original: IUserTableData;
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
  id?: string;
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
  teamNames: string[];
  roleGroups: { role: string; names: string[] }[];
  role: UserRole;
  actions: IDropdownOption[];
  /** Prefixed ID used as a unique react-table row key (e.g. "user-3", "invite-1") */
  id: string;
  /** Numeric ID used for API calls */
  apiId: number;
  type: string;
  api_only: boolean;
  /** Number of API endpoints this user is restricted to; 0 means unrestricted */
  apiEndpointCount: number;
}

// The inactivity window is enforced server-side (see UserInactiveAfter in
// server/fleet/users.go); these tooltips just describe it.
const USER_INACTIVE_TOOLTIP = "Hasn't logged in for 30+ days";
const API_ONLY_INACTIVE_TOOLTIP = "No API activity for 30+ days";

const USER_STATUS_DISPLAY_TEXT: Record<UserStatus, string> = {
  active: "Active",
  inactive: "Inactive",
  no_access: "No access",
};

/** The user's status is computed server-side ("active" | "inactive" |
 * "no_access") and only mapped to display text here. */
const generateUserStatus = (user: IUser): string =>
  USER_STATUS_DISPLAY_TEXT[user.status ?? "active"];

// Invites come from a separate endpoint and have no server-computed status,
// so their status stays derived client-side.
const generateInviteStatus = (invite: IInvite): string =>
  invite.global_role === null && invite.teams.length === 0
    ? "No access"
    : "Invite pending";

const renderRole = (cellProps: ICellProps) => {
  if (cellProps.cell.value === "GitOps") {
    return (
      <TooltipWrapper
        tipContent={
          <>
            The GitOps role is only available for API-only
            <br />
            users. This user has no access to the UI.
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
  if (cellProps.cell.value === ROLE_VARIOUS) {
    const { roleGroups } = cellProps.row.original;
    return (
      <TooltipWrapper
        tipContent={roleGroups.map(({ role, names }) => (
          <span key={role}>
            <b>{role}:</b> {names.join(", ")}
            <br />
          </span>
        ))}
        underline={false}
        showArrow
        position="top"
        tipOffset={10}
        fixedPositionStrategy
      >
        <TextCell value={ROLE_VARIOUS} grey italic />
      </TooltipWrapper>
    );
  }
  return (
    <TextCell
      value={cellProps.cell.value}
      grey={greyCell(cellProps.cell.value)}
      italic={greyCell(cellProps.cell.value)}
    />
  );
};

const generateTableHeaders = (
  actionSelectHandler: (value: string, user: IUserTableData) => void,
  isPremiumTier: boolean | undefined
): IDataColumn[] => {
  const tableHeaders: IDataColumn[] = [
    {
      title: "Name",
      Header: "Name",
      disableSortBy: true,
      accessor: "name",
      Cell: (cellProps: ICellProps) => {
        const apiOnlyUser =
          "api_only" in cellProps.row.original
            ? cellProps.row.original.api_only
            : false;

        return (
          <TooltipTruncatedTextCell
            value={cellProps.cell.value}
            suffix={apiOnlyUser && renderApiUserIndicator()}
          />
        );
      },
    },
    {
      title: "Permissions",
      Header: "Permissions",
      accessor: "role",
      // react-table derives the cell/header DOM classes and the sort key from
      // `id`, so this keeps them as `permissions__*` without renaming the
      // underlying row field.
      id: "permissions",
      disableSortBy: true,
      Cell: (cellProps: ICellProps) => {
        const { apiEndpointCount } = cellProps.row.original;

        return (
          <div className={`${baseClass}__permissions-content`}>
            {renderRole(cellProps)}
            {apiEndpointCount > 0 && renderApiEndpointCount(apiEndpointCount)}
          </div>
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
        <StatusIndicator
          value={cellProps.cell.value}
          tooltip={
            cellProps.cell.value === "Inactive"
              ? {
                  tooltipText: cellProps.row.original.api_only
                    ? API_ONLY_INACTIVE_TOOLTIP
                    : USER_INACTIVE_TOOLTIP,
                }
              : undefined
          }
        />
      ),
    },
    {
      title: "Email",
      Header: "Email",
      disableSortBy: true,
      accessor: "email",
      Cell: (cellProps: ICellProps) => {
        const isApiOnly = cellProps.row.original.api_only;
        if (isApiOnly) {
          return (
            <TooltipWrapper
              tipContent="API-only users do not receive emails or log into the UI."
              underline={false}
            >
              ---
            </TooltipWrapper>
          );
        }
        return <TextCell value={cellProps.cell.value} />;
      },
    },
    {
      title: "Actions",
      Header: "",
      disableSortBy: true,
      accessor: "actions",
      Cell: (cellProps: IActionsDropdownProps) => (
        <div
          role="presentation"
          onClick={(e) => e.stopPropagation()}
          onKeyDown={(e) => e.stopPropagation()}
        >
          <ActionsDropdown
            className="row-hover-button"
            options={cellProps.cell.value}
            onChange={(value: string) =>
              actionSelectHandler(value, cellProps.row.original)
            }
            placeholder="Actions"
            menuAlign="right"
            variant="secondary"
          />
        </div>
      ),
    },
  ];

  // Add Teams column for premium tier
  if (isPremiumTier) {
    tableHeaders.splice(2, 0, {
      title: "Fleets",
      Header: "Fleets",
      accessor: "teams",
      disableSortBy: true,
      Cell: (cellProps: ICellProps) => {
        const { teamNames } = cellProps.row.original;
        if (teamNames.length > 1) {
          return (
            <TooltipWrapper
              tipContent={tooltipTextWithLineBreaks(teamNames)}
              underline={false}
              showArrow
              position="top"
              tipOffset={10}
              fixedPositionStrategy
            >
              <TextCell value={cellProps.cell.value} grey italic />
            </TooltipWrapper>
          );
        }
        const isGrey = greyCell(cellProps.cell.value);
        return (
          <TextCell
            value={cellProps.cell.value}
            grey={isGrey}
            italic={isGrey && cellProps.cell.value !== ROLE_GLOBAL}
          />
        );
      },
    });
  }

  return tableHeaders;
};

const generateActionDropdownOptions = (
  isCurrentUser: boolean,
  isInvitePending: boolean,
  isSsoEnabled: boolean,
  isApiOnly: boolean
): IDropdownOption[] => {
  const disableDelete = isCurrentUser;

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
      disabled: disableDelete,
      value: "delete",
      tooltipContent: disableDelete ? (
        <>
          There must be at least one Admin
          <br />
          user on the account. To delete this
          <br />
          user, add or set existing user with
          <br />
          role of &quot;Admin&quot;.
        </>
      ) : undefined,
    },
  ];

  if (isCurrentUser) {
    // remove "Reset sessions" from dropdownOptions
    dropdownOptions = dropdownOptions.filter(
      (option) => option.label !== "Reset sessions"
    );
  }

  if (isSsoEnabled || isApiOnly) {
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
      status: generateUserStatus(user),
      email: user.email,
      teams: generateTeam(user.teams, user.global_role),
      teamNames: generateTeamNames(user.teams),
      roleGroups: generateRoleGroups(user.teams),
      role: generateRole(user.teams, user.global_role),
      actions: generateActionDropdownOptions(
        user.id === currentUserId,
        false,
        user.sso_enabled,
        user.api_only
      ),
      id: `user-${user.id}`,
      apiId: user.id,
      type: "user",
      api_only: user.api_only,
      apiEndpointCount: user.api_endpoints?.length ?? 0,
    };
  });
};

const enhanceInviteData = (invites: IInvite[]): IUserTableData[] => {
  return invites.map((invite) => {
    return {
      name: invite.name || DEFAULT_EMPTY_CELL_VALUE,
      status: generateInviteStatus(invite),
      email: invite.email,
      teams: generateTeam(invite.teams, invite.global_role),
      teamNames: generateTeamNames(invite.teams),
      roleGroups: generateRoleGroups(invite.teams),
      role: generateRole(invite.teams, invite.global_role),
      actions: generateActionDropdownOptions(
        false,
        true,
        invite.sso_enabled,
        false
      ),
      id: `invite-${invite.id}`,
      apiId: invite.id,
      type: "invite",
      api_only: false, // api only users are created through fleetctl and not invites
      apiEndpointCount: 0,
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
