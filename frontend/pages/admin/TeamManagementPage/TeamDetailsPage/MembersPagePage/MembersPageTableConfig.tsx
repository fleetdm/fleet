import React from "react";
import TextCell from "components/TableContainer/DataTable/TextCell/TextCell";
import DropdownCell from "components/TableContainer/DataTable/DropdownCell";
import { IUser } from "interfaces/user";
import { ITeam } from "interfaces/team";
import { IDropdownOption } from "interfaces/dropdownOption";

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
      Header: "role",
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

const generateActionDropdownOptions = (id: number): IDropdownOption[] => {
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
const generateRole = (teams: ITeam[], globalRole: string | null): string => {
  if (globalRole === null) {
    if (teams.length === 0) {
      // no global role and no teams
      return "Unassigned";
    } else if (teams.length === 1) {
      // no global role and only one team
      return teams[0].role as string;
    }
    return "Various"; // no global role and multiple teams
  }

  if (teams.length === 0) {
    // global role and no teams
    return globalRole;
  }
  return "Various"; // global role and one or more teams
};

const enhanceMembersData = (users: {
  [id: number]: IUser;
}): IMembersTableData[] => {
  return Object.values(users).map((user) => {
    return {
      name: user.name,
      email: user.email,
      role: generateRole(user.teams, user.global_role),
      actions: generateActionDropdownOptions(user.id),
      id: user.id,
    };
  });
};

const generateDataSet = (users: {
  [id: number]: IUser;
}): IMembersTableData[] => {
  return [...enhanceMembersData(users)];
};

export { generateTableHeaders, generateDataSet };
