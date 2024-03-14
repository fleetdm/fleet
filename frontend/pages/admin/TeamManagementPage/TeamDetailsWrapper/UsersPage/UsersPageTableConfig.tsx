import React from "react";
import ReactTooltip from "react-tooltip";
import TextCell from "components/TableContainer/DataTable/TextCell/TextCell";
import DropdownCell from "components/TableContainer/DataTable/DropdownCell";
import CustomLink from "components/CustomLink";
import { IUser, UserRole } from "interfaces/user";
import { ITeam } from "interfaces/team";
import { IDropdownOption } from "interfaces/dropdownOption";
import stringUtils from "utilities/strings";
import TooltipWrapper from "components/TooltipWrapper";
import { COLORS } from "styles/var/colors";

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
  sortType?: string;
}

export interface ITeamUsersTableData {
  name: string;
  email: string;
  role: UserRole;
  teams: ITeam[];
  actions: IDropdownOption[];
  id: number;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateColumnConfigs = (
  actionSelectHandler: (value: string, user: IUser) => void
): IDataColumn[] => {
  return [
    {
      title: "Name",
      Header: "Name",
      disableSortBy: true,
      sortType: "caseInsensitive",
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
                    className="team-users__api-only-user"
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
      disableSortBy: true,
      accessor: "role",
      Cell: (cellProps: ICellProps) => {
        if (cellProps.cell.value === "GitOps") {
          return (
            <TooltipWrapper
              position="top-start"
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
              position="top-start"
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
        return <TextCell value={cellProps.cell.value} />;
      },
    },
    {
      title: "Email",
      Header: "Email",
      disableSortBy: true,
      accessor: "email",
      Cell: (cellProps: ICellProps) => (
        <TextCell classes="w400" value={cellProps.cell.value} />
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
          placeholder="Actions"
        />
      ),
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
      global_role: user.global_role,
      actions: generateActionDropdownOptions(),
      id: user.id,
      api_only: user.api_only,
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
