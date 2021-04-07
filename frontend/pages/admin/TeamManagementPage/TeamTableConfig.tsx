import React from 'react';

import StatusCell from 'components/TableContainer/DataTable/StatusCell/StatusCell';
import TextCell from 'components/TableContainer/DataTable/TextCell/TextCell';
import DropdownCell from 'components/TableContainer/DataTable/DropdownCell';
import { ITeam } from 'interfaces/team';
import { IDropdownOption } from 'interfaces/dropdownOption';

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  }
}

interface ICellProps {
  cell: {
    value: any;
  };
  row: {
    original: ITeam;
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

interface ITeamTableData {
  name: string;
  hosts: number;
  members: number;
  actions: IDropdownOption[];
  id: number;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateTableHeaders = (actionSelectHandler: (value: string, team: ITeam) => void): IDataColumn[] => {
  return [
    {
      title: 'Name',
      Header: 'Name',
      disableSortBy: true,
      accessor: 'name',
      Cell: cellProps => <TextCell value={cellProps.cell.value} />,
    },
    // TODO: need to add this info to API
    {
      title: 'Hosts',
      Header: 'Hosts',
      disableSortBy: true,
      accessor: 'hosts',
      Cell: cellProps => <TextCell value={cellProps.cell.value} />,
    },
    {
      title: 'Members',
      Header: 'Members',
      disableSortBy: true,
      accessor: 'members',
      Cell: cellProps => <TextCell value={cellProps.cell.value} />,
    },
    {
      title: 'Actions',
      Header: 'Actions',
      disableSortBy: true,
      accessor: 'actions',
      Cell: cellProps => (
        <DropdownCell
          options={cellProps.cell.value}
          onChange={(value: string) => actionSelectHandler(value, cellProps.row.original)}
          placeholder={'Actions'}
        />
      ),
    },
  ];
};

// NOTE: may need current user ID later for permission on actions.
const generateActionDropdownOptions = (): IDropdownOption[] => {
  return [
    {
      label: 'Edit',
      disabled: false,
      value: 'edit',
    },
    {
      label: 'Delete',
      disabled: false,
      value: 'delete',
    },
  ];
};

const enhanceTeamData = (teams: {[id: number]: ITeam}): ITeamTableData[] => {
  return Object.values(teams).map((team) => {
    return {
      name: team.name,
      hosts: team.hosts,
      members: team.members,
      actions: generateActionDropdownOptions(),
      id: team.id,
    };
  });
};

const generateDataSet = (teams: {[id: number]: ITeam}): ITeamTableData[] => {
  return [...enhanceTeamData(teams)];
};

export { generateTableHeaders, generateDataSet };
