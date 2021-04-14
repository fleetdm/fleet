import React from "react";
import { IHost } from "../../../../../interfaces/host";

import HeaderCell from "../HeaderCell/HeaderCell";
import LinkCell from "../LinkCell/LinkCell";
import StatusCell from "../StatusCell/StatusCell";
import TextCell from "../TextCell/TextCell";
import {
  humanHostMemory,
  humanHostUptime,
  humanHostLastSeen,
  humanHostDetailUpdated,
} from "../../../../../kolide/helpers";

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

interface ICellProps {
  cell: {
    value: string;
  };
  row: {
    original: IHost;
  };
}

interface IHostDataColumn {
  title: string;
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  accessor: string;
  Cell: (props: ICellProps) => JSX.Element;
  disableHidden?: boolean;
  disableSortBy?: boolean;
}

const hostDataHeaders: IHostDataColumn[] = [
  {
    title: "Hostname",
    Header: (cellProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "hostname",
    Cell: (cellProps) => (
      <LinkCell value={cellProps.cell.value} host={cellProps.row.original} />
    ),
    disableHidden: true,
  },
  {
    title: "Status",
    Header: "Status",
    disableSortBy: true,
    accessor: "status",
    Cell: (cellProps) => <StatusCell value={cellProps.cell.value} />,
  },
  {
    title: "OS",
    Header: (cellProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "os_version",
    Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Osquery",
    Header: (cellProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "osquery_version",
    Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "IP address",
    Header: (cellProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "primary_ip",
    Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Last fetched",
    Header: (cellProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "detail_updated_at",
    Cell: (cellProps) => (
      <TextCell
        value={cellProps.cell.value}
        formatter={humanHostDetailUpdated}
      />
    ),
  },
  {
    title: "Last seen",
    Header: (cellProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "seen_time",
    Cell: (cellProps) => (
      <TextCell value={cellProps.cell.value} formatter={humanHostLastSeen} />
    ),
  },
  {
    title: "UUID",
    Header: (cellProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "uuid",
    Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Uptime",
    Header: (cellProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "uptime",
    Cell: (cellProps) => (
      <TextCell value={cellProps.cell.value} formatter={humanHostUptime} />
    ),
  },
  {
    title: "CPU",
    Header: "CPU",
    disableSortBy: true,
    accessor: "host_cpu",
    Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "RAM",
    Header: (cellProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "memory",
    Cell: (cellProps) => (
      <TextCell value={cellProps.cell.value} formatter={humanHostMemory} />
    ),
  },
  {
    title: "MAC address",
    Header: (cellProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "primary_mac",
    Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Serial Number",
    Header: (cellProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "hardware_serial",
    Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Hardware model",
    Header: (cellProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "hardware_model",
    Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
  },
];

const defaultHiddenColumns = [
  "primary_mac",
  "host_cpu",
  "memory",
  "uptime",
  "uuid",
  "seen_time",
  "hardware_model",
  "hardware_serial",
];

export { hostDataHeaders, defaultHiddenColumns };
