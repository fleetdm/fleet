import React from 'react';

import HeaderCell from '../HeaderCell/HeaderCell';
import LinkCell from '../LinkCell/LinkCell';
import StatusCell from '../StatusCell/StatusCell';
import TextCell from '../TextCell/TextCell';
import { humanHostMemory, humanHostUptime } from '../../../../../kolide/helpers';

const hostDataHeaders = [
  {
    title: 'Hostname',
    Header: cellProps => <HeaderCell value={cellProps.column.title} isSortedDesc={cellProps.column.isSortedDesc} />,
    accessor: 'hostname',
    Cell: cellProps => <LinkCell value={cellProps.cell.value} host={cellProps.row.original} />,
    canHide: false,
  },
  {
    title: 'Status',
    Header: 'Status',
    disableSortBy: true,
    accessor: 'status',
    Cell: cellProps => <StatusCell value={cellProps.cell.value} />,
  },
  {
    title: 'OS',
    Header: cellProps => <HeaderCell all={cellProps.column} value={cellProps.column.title} isSortedDesc={cellProps.column.isSortedDesc} />,
    accessor: 'os_version',
    Cell: cellProps => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: 'Osquery',
    Header: cellProps => <HeaderCell value={cellProps.column.title} isSortedDesc={cellProps.column.isSortedDesc} />,
    accessor: 'osquery_version',
    Cell: cellProps => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: 'IPv4',
    Header: cellProps => <HeaderCell value={cellProps.column.title} isSortedDesc={cellProps.column.isSortedDesc} />,
    accessor: 'primary_ip',
    Cell: cellProps => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: 'Physical Address',
    Header: cellProps => <HeaderCell value={cellProps.column.title} isSortedDesc={cellProps.column.isSortedDesc} />,
    accessor: 'primary_mac',
    Cell: cellProps => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: 'CPU',
    Header: 'CPU',
    disableSortBy: true,
    accessor: 'host_cpu',
    Cell: cellProps => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: 'Memory',
    Header: cellProps => <HeaderCell value={cellProps.column.title} isSortedDesc={cellProps.column.isSortedDesc} />,
    accessor: 'memory',
    Cell: cellProps => <TextCell value={cellProps.cell.value} formatter={humanHostMemory} />,
  },
  {
    title: 'Uptime',
    Header: cellProps => <HeaderCell value={cellProps.column.title} isSortedDesc={cellProps.column.isSortedDesc} />,
    accessor: 'uptime',
    Cell: cellProps => <TextCell value={cellProps.cell.value} formatter={humanHostUptime} />,
  },
  {
    title: 'UUID',
    Header: cellProps => <HeaderCell value={cellProps.column.title} isSortedDesc={cellProps.column.isSortedDesc} />,
    accessor: 'uuid',
    Cell: cellProps => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: 'Seen Time',
    Header: cellProps => <HeaderCell value={cellProps.column.title} isSortedDesc={cellProps.column.isSortedDesc} />,
    accessor: 'seen_time',
    Cell: cellProps => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: 'Hardware Model',
    Header: cellProps => <HeaderCell value={cellProps.column.title} isSortedDesc={cellProps.column.isSortedDesc} />,
    accessor: 'hardware_model',
    Cell: cellProps => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: 'Hardware Serial',
    Header: cellProps => <HeaderCell value={cellProps.column.title} isSortedDesc={cellProps.column.isSortedDesc} />,
    accessor: 'hardware_serial',
    Cell: cellProps => <TextCell value={cellProps.cell.value} />,
  },
];

const defaultHiddenColumns = [
  'primary_mac',
  'host_cpu',
  'memory',
  'uptime',
  'uuid',
  'seen_time',
  'hardware_model',
  'hardware_serial',
];

export { hostDataHeaders, defaultHiddenColumns };
