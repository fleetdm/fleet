import React from "react";

import { IDiskEncryptionStatusAggregate } from "interfaces/mdm";

import TextCell from "components/TableContainer/DataTable/TextCell";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import StatusIndicatorWithIcon from "components/StatusIndicatorWithIcon";

interface IStatusCellValue {
  displayName: string;
  statusName: "success" | "pending" | "error";
  tooltip?: string | JSX.Element;
}

interface IStatusCellProps {
  cell: {
    value: IStatusCellValue;
  };
}

interface ICellProps {
  cell: {
    value: string;
  };
}

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

type CellWithCellProps = {
  Cell: (props: ICellProps) => JSX.Element;
};

type CellWithStatusCellProps = {
  Cell: (props: IStatusCellProps) => JSX.Element;
};

type IDataColumn = {
  title: string;
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  accessor: string;
  disableHidden?: boolean;
  disableSortBy?: boolean;
} & (CellWithCellProps | CellWithStatusCellProps);

const defaultTableHeaders: IDataColumn[] = [
  {
    title: "Status",
    Header: "Status",
    disableSortBy: true,
    accessor: "status",
    Cell: ({ cell: { value } }: IStatusCellProps) => {
      const tooltipProp = value.tooltip
        ? { tooltipText: value.tooltip }
        : undefined;
      return (
        <StatusIndicatorWithIcon
          status={value.statusName}
          value={value.displayName}
          tooltip={tooltipProp}
        />
      );
    },
  },
  {
    title: "Hosts",
    Header: (cellProps: IHeaderProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
        disableSortBy={false}
      />
    ),
    disableSortBy: true,
    accessor: "hosts",
    Cell: ({ cell: { value } }: ICellProps) => <TextCell value={value} />,
  },
];

type StatusNames = keyof IDiskEncryptionStatusAggregate;

type StatusEntry = [StatusNames, number];

export const generateTableHeaders = (): IDataColumn[] => {
  return defaultTableHeaders;
};

const STATUS_CELL_VALUES: Record<StatusNames, IStatusCellValue> = {
  applied: {
    displayName: "Applied",
    statusName: "success",
    tooltip: "Disk encryption on and key stored in Fleet.",
  },
  action_required: {
    displayName: "Action required (pending)",
    statusName: "pending",
    tooltip: (
      <>
        Ask the end user to follow <b>Disk encryption</b> instructions on their{" "}
        <b>My device</b> page.
      </>
    ),
  },
  enforcing: {
    displayName: "Enforcing (pending)",
    statusName: "pending",
    tooltip: "Setting will be enforced when the hosts come online.",
  },
  failed: {
    displayName: "Failed",
    statusName: "error",
  },
  removing_enforcement: {
    displayName: "Removing enforcement (pending)",
    statusName: "pending",
    tooltip: "Enforcement will be removed when the hosts come online.",
  },
};

export const generateTableData = (data: IDiskEncryptionStatusAggregate) => {
  const entries = Object.entries(data) as StatusEntry[];

  return entries.map(([status, numHosts]) => ({
    // eslint-disable-next-line object-shorthand
    status: STATUS_CELL_VALUES[status],
    hosts: numHosts,
  }));
};
