/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React from "react";

import { IHost } from "interfaces/host";
// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import Checkbox from "components/forms/fields/Checkbox";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import LinkCell from "components/TableContainer/DataTable/LinkCell/LinkCell";
import StatusCell from "components/TableContainer/DataTable/StatusCell/StatusCell";
import TextCell from "components/TableContainer/DataTable/TextCell/TextCell";
import {
  humanHostMemory,
  humanHostUptime,
  humanHostLastSeen,
  humanHostDetailUpdated,
} from "kolide/helpers";
import PATHS from "../../../router/paths";

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
    value: string;
  };
  row: {
    original: IHost;
    getToggleRowSelectedProps: () => any; // TODO: do better with types
    toggleRowSelected: () => void;
  };
}

interface IHostDataColumn {
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  Cell: (props: ICellProps) => JSX.Element;
  id?: string;
  title?: string;
  accessor?: string;
  disableHidden?: boolean;
  disableSortBy?: boolean;
}

const lastSeenTime = (status: string, seenTime: string): string => {
  if (status !== "online") {
    return `Last Seen: ${humanHostLastSeen(seenTime)} UTC`;
  }
  return "Online";
};

const hostTableHeaders: IHostDataColumn[] = [
  // We are using React Table useRowSelect functionality for the selection header.
  // More information on its API can be found here
  // https://react-table.tanstack.com/docs/api/useRowSelect
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
    title: "Hostname",
    Header: (cellProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "hostname",
    Cell: (cellProps) => (
      <LinkCell
        value={cellProps.cell.value}
        path={PATHS.HOST_DETAILS(cellProps.row.original)}
        title={lastSeenTime(
          cellProps.row.original.status,
          cellProps.row.original.seen_time
        )}
      />
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

const generateVisibleHostColumns = (
  hiddenColumns: string[]
): IHostDataColumn[] => {
  return hostTableHeaders.filter((column) => {
    return !hiddenColumns.includes(column.accessor as string);
  });
};

export { hostTableHeaders, defaultHiddenColumns, generateVisibleHostColumns };
