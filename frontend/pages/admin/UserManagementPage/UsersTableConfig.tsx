import React from 'react';

import HeaderCell from 'components/TableContainer/DataTable/HeaderCell/HeaderCell';
// import StatusCell from 'components/DataTable/StatusCell/StatusCell';
import TextCell from 'components/TableContainer/DataTable/TextCell/TextCell';
import { IUser } from 'interfaces/user';

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  }
}

interface ICellProps {
  cell: {
    value: string;
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

const usersTableHeaders: IDataColumn[] = [
  {
    title: 'Name',
    Header: cellProps => <HeaderCell value={cellProps.column.title} isSortedDesc={cellProps.column.isSortedDesc} />,
    accessor: 'name',
    Cell: cellProps => <TextCell value={cellProps.cell.value} />,
  },
  // TODO: need to add this info to API
  // {
  //   title: 'Status',
  //   Header: 'Status',
  //   accessor: 'status',
  //   Cell: cellProps => <StatusCell value={cellProps.cell.value} />,
  // },
  {
    title: 'Email',
    Header: cellProps => <HeaderCell value={cellProps.column.title} isSortedDesc={cellProps.column.isSortedDesc} />,
    accessor: 'email',
    Cell: cellProps => <TextCell value={cellProps.cell.value} />,
  },
  // TODO: need to add this info to API
  // {
  //   title: 'Teams',
  //   Header: cellProps => <HeaderCell value={cellProps.column.title} isSortedDesc={cellProps.column.isSortedDesc} />,
  //   accessor: 'osquery_version',
  //   Cell: cellProps => <TextCell value={cellProps.cell.value} />,
  // },
  // TODO: need to add this info to API
  // {
  //   title: 'Roles',
  //   Header: cellProps => <HeaderCell value={cellProps.column.title} isSortedDesc={cellProps.column.isSortedDesc} />,
  //   accessor: 'primary_ip',
  //   Cell: cellProps => <TextCell value={cellProps.cell.value} />,
  // },
  // TODO: figure out this column accessor
  // {
  //   title: 'Actions',
  //   Header: cellProps => <HeaderCell value={cellProps.column.title} isSortedDesc={cellProps.column.isSortedDesc} />,
  //   accessor: 'actions',
  //   Cell: cellProps => <TextCell value={cellProps.cell.value} formatter={humanHostLastSeen} />,
  // },
];

export default usersTableHeaders;
