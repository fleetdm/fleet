import React from "react";

import LinkCell from "components/TableContainer/DataTable/LinkCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import ActionsDropdown from "components/ActionsDropdown";
import { ITeam } from "interfaces/team";
import { IDropdownOption } from "interfaces/dropdownOption";
import PATHS from "router/paths";

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

interface IRowProps {
  row: {
    original: ITeam;
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
  sortType?: string;
}

interface ITeamTableData extends ITeam {
  actions: IDropdownOption[];
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateTableHeaders = (
  actionSelectHandler: (value: string, team: ITeam) => void
): IDataColumn[] => {
  return [
    {
      title: "Name",
      Header: "Name",
      disableSortBy: true,
      sortType: "caseInsensitive",
      accessor: "name",
      Cell: (cellProps: ICellProps) => (
        <LinkCell
          value={cellProps.cell.value}
          path={PATHS.TEAM_DETAILS_USERS(cellProps.row.original.id)}
        />
      ),
    },
    // TODO: need to add this info to API
    {
      title: "Hosts",
      Header: "Hosts",
      disableSortBy: true,
      accessor: "host_count",
      Cell: (cellProps: ICellProps) => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      title: "Users",
      Header: "Users",
      disableSortBy: true,
      accessor: "user_count",
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
        <ActionsDropdown
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

// NOTE: may need current user ID later for permission on actions.
const generateActionDropdownOptions = (): IDropdownOption[] => {
  return [
    {
      label: "Rename",
      disabled: false,
      value: "rename",
    },
    {
      label: "Delete",
      disabled: false,
      value: "delete",
    },
  ];
};

const enhanceTeamData = (teams: ITeam[]): ITeamTableData[] => {
  return Object.values(teams).map((team) => {
    return {
      description: team.description,
      name: team.name,
      host_count: team.host_count,
      user_count: team.user_count,
      actions: generateActionDropdownOptions(),
      id: team.id,
    };
  });
};

const generateDataSet = (teams: ITeam[]): ITeamTableData[] => {
  return [...enhanceTeamData(teams)];
};

export { generateTableHeaders, generateDataSet };
