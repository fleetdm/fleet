import React from "react";
import TextCell from "components/TableContainer/DataTable/TextCell/TextCell";
import DropdownCell from "components/TableContainer/DataTable/DropdownCell";
import { IUser } from "interfaces/user";
import { ITeam } from "interfaces/team";
import { IDropdownOption } from "interfaces/dropdownOption";
import stringUtils from "utilities/strings";

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

interface ICellProps {
  cell: {
    value: any;
  };
  row: {
    original: IUser;
  };
}

interface IDataColumn {
  title: string;
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  accessor: string;
  Cell: (props: ICellProps) => JSX.Element;
  disableHidden?: boolean;
  disableSortBy?: boolean;
}

interface IMembersTableData {
  name: string;
  email: string;
  role: string;
  teams: ITeam[];
  actions: IDropdownOption[];
  id: number;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateTableHeaders = (
  actionSelectHandler: (value: string, user: IUser) => void
): IDataColumn[] => {
  return [
    {
      title: "Name",
      Header: "Name",
      disableSortBy: true,
      accessor: "name",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      title: "Email",
      Header: "Email",
      disableSortBy: true,
      accessor: "email",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      title: "Role",
      Header: "Role",
      disableSortBy: true,
      accessor: "role",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      title: "Actions",
      Header: "Actions",
      disableSortBy: true,
      accessor: "actions",
      Cell: (cellProps) => (
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
const generateRole = (teamId: number, teams: ITeam[]): string => {
  const role = teams.find((team) => teamId === team.id)?.role ?? "";
  return stringUtils.capitalize(role);
};

const enhanceMembersData = (
  teamId: number,
  users: {
    [id: number]: IUser;
  }
): IMembersTableData[] => {
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
    };
  });
};

const generateDataSet = (
  teamId: number,
  users: {
    [id: number]: IUser;
  }
): IMembersTableData[] => {
  return [...enhanceMembersData(teamId, users)];
};

export { generateTableHeaders, generateDataSet };
