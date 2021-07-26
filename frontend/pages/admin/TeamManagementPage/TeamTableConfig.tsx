import React from "react";

import Checkbox from "components/forms/fields/Checkbox";
import LinkCell from "components/TableContainer/DataTable/LinkCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import DropdownCell from "components/TableContainer/DataTable/DropdownCell";
import { ITeam } from "interfaces/team";
import { IDropdownOption } from "interfaces/dropdownOption";
import PATHS from "router/paths";

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
  getToggleAllRowsSelectedProps: () => any; // TODO: do better with types
  toggleAllRowsSelected: () => void;
}

interface ICellProps {
  cell: {
    value: any;
  };
  row: {
    original: ITeam;
    getToggleRowSelectedProps: () => any; // TODO: do better with types
    toggleRowSelected: () => void;
  };
}

interface IDataColumn {
  id?: string;
  title?: string;
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  accessor?: string;
  Cell: (props: ICellProps) => JSX.Element;
  disableHidden?: boolean;
  disableSortBy?: boolean;
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
      id: "selection",
      Header: (cellProps: IHeaderProps): JSX.Element => {
        const props = cellProps.getToggleAllRowsSelectedProps();
        const checkboxProps = {
          value: props.checked,
          indeterminate: props.indeterminate,
          onChange: () => cellProps.toggleAllRowsSelected(),
        };
        return <Checkbox {...checkboxProps} />;
      },
      Cell: (cellProps: ICellProps): JSX.Element => {
        const props = cellProps.row.getToggleRowSelectedProps();
        const checkboxProps = {
          value: props.checked,
          onChange: () => cellProps.row.toggleRowSelected(),
        };
        return <Checkbox {...checkboxProps} />;
      },
      disableHidden: true,
    },
    {
      title: "Name",
      Header: "Name",
      disableSortBy: true,
      accessor: "name",
      Cell: (cellProps) => (
        <LinkCell
          value={cellProps.cell.value}
          path={PATHS.TEAM_DETAILS_MEMBERS(cellProps.row.original.id)}
        />
      ),
    },
    // TODO: need to add this info to API
    {
      title: "Hosts",
      Header: "Hosts",
      disableSortBy: true,
      accessor: "host_count",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      title: "Members",
      Header: "Members",
      disableSortBy: true,
      accessor: "user_count",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      title: "Actions",
      Header: "",
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

// NOTE: may need current user ID later for permission on actions.
const generateActionDropdownOptions = (): IDropdownOption[] => {
  return [
    {
      label: "Edit",
      disabled: false,
      value: "edit",
    },
    {
      label: "Delete",
      disabled: false,
      value: "delete",
    },
  ];
};

const enhanceTeamData = (teams: { [id: number]: ITeam }): ITeamTableData[] => {
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

const generateDataSet = (teams: { [id: number]: ITeam }): ITeamTableData[] => {
  return [...enhanceTeamData(teams)];
};

export { generateTableHeaders, generateDataSet };
