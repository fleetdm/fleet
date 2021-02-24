import React from 'react';

import HeaderCell from '../HeaderCell/HeaderCell';
import LinkCell from '../LinkCell/LinkCell';
import StatusCell from '../StatusCell/StatusCell';
import TextCell from '../TextCell/TextCell';
import { humanHostMemory, humanHostUptime } from '../../../../../kolide/helpers';

const hostDataHeaders = [
  {
    Header: cellProps => <HeaderCell value={'Hostname'} isSortedDesc={cellProps.column.isSortedDesc} />,
    accessor: 'hostname',
    Cell: cellProps => <LinkCell value={cellProps.cell.value} host={cellProps.row.original} />,
    canHide: false,
  },
  {
    Header: 'Status',
    disableSortBy: true,
    accessor: 'status',
    Cell: cellProps => <StatusCell value={cellProps.cell.value} />,
  },
  {
    Header: cellProps => <HeaderCell all={cellProps.column} value={'OS'} isSortedDesc={cellProps.column.isSortedDesc} />,
    accessor: 'os_version',
    Cell: cellProps => <TextCell value={cellProps.cell.value} />,
  },
  {
    Header: cellProps => <HeaderCell value={'Osquery'} isSortedDesc={cellProps.column.isSortedDesc} />,
    accessor: 'osquery_version',
    Cell: cellProps => <TextCell value={cellProps.cell.value} />,
  },
  {
    Header: cellProps => <HeaderCell value={'IPv4'} isSortedDesc={cellProps.column.isSortedDesc} />,
    accessor: 'primary_ip',
    Cell: cellProps => <TextCell value={cellProps.cell.value} />,
  },
  {
    Header: cellProps => <HeaderCell value={'Physical Address'} isSortedDesc={cellProps.column.isSortedDesc} />,
    accessor: 'primary_mac',
    Cell: cellProps => <TextCell value={cellProps.cell.value} />,
  },
  {
    Header: 'CPU',
    disableSortBy: true,
    accessor: 'host_cpu',
    Cell: cellProps => <TextCell value={cellProps.cell.value} />,
  },
  {
    Header: cellProps => <HeaderCell value={'Memory'} isSortedDesc={cellProps.column.isSortedDesc} />,
    accessor: 'memory',
    Cell: cellProps => <TextCell value={cellProps.cell.value} formatter={humanHostMemory} />,
  },
  {
    Header: cellProps => <HeaderCell value={'Uptime'} isSortedDesc={cellProps.column.isSortedDesc} />,
    accessor: 'uptime',
    Cell: cellProps => <TextCell value={cellProps.cell.value} formatter={humanHostUptime} />,
  },
  {
    Header: cellProps => <HeaderCell value={'UUID'} isSortedDesc={cellProps.column.isSortedDesc} />,
    accessor: 'uuid',
    Cell: cellProps => <TextCell value={cellProps.cell.value} />,
  },
  {
    Header: cellProps => <HeaderCell value={'Seen Time'} isSortedDesc={cellProps.column.isSortedDesc} />,
    accessor: 'seen_time',
    Cell: cellProps => <TextCell value={cellProps.cell.value} />,
  },
  {
    Header: cellProps => <HeaderCell value={'Hardware Model'} isSortedDesc={cellProps.column.isSortedDesc} />,
    accessor: 'hardware_model',
    Cell: cellProps => <TextCell value={cellProps.cell.value} />,
  },
  {
    Header: cellProps => <HeaderCell value={'Hardware Serial'} isSortedDesc={cellProps.column.isSortedDesc} />,
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
