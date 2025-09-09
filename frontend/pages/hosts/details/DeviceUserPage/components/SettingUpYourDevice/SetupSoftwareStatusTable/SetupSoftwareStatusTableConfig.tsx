import React from "react";

import { CellProps, Column } from "react-table";

import { ISetupSoftwareStatus } from "interfaces/software";

import SetupSoftwareProcessCell from "components/TableContainer/DataTable/SetupSoftwareProcessCell";
import SetupSoftwareStatusCell from "components/TableContainer/DataTable/SetupSoftwareStatusCell";

type ISetupSoftwareStatusTableConfig = Column<ISetupSoftwareStatus>;
type ITableCellProps = CellProps<ISetupSoftwareStatus>;

const generateColumnConfigs = (): ISetupSoftwareStatusTableConfig[] => [
  {
    Header: "Process",
    accessor: "name",
    disableSortBy: true,
    Cell: (cellProps: ITableCellProps) => {
      const { name } = cellProps.row.original;

      return <SetupSoftwareProcessCell name={name || "Unknown software"} />;
    },
  },
  {
    Header: "Status",
    accessor: "status",
    disableSortBy: true,
    Cell: (cellProps: ITableCellProps) => {
      const { status } = cellProps.row.original;

      return <SetupSoftwareStatusCell status={status || "pending"} />;
    },
  },
];

export default generateColumnConfigs;
