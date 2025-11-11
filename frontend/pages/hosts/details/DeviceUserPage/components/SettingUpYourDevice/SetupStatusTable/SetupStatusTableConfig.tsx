import React from "react";

import { CellProps, Column } from "react-table";

import { ISetupStep } from "interfaces/setup";

import SetupSoftwareProcessCell from "components/TableContainer/DataTable/SetupSoftwareProcessCell";
import SetupSoftwareStatusCell from "components/TableContainer/DataTable/SetupSoftwareStatusCell";
import SetupScriptProcessCell from "components/TableContainer/DataTable/SetupScriptProcessCell";
import SetupScriptStatusCell from "components/TableContainer/DataTable/SetupScriptStatusCell";

type ISetupStatusTableConfig = Column<ISetupStep>;
type ITableCellProps = CellProps<ISetupStep>;

const generateColumnConfigs = (): ISetupStatusTableConfig[] => [
  {
    Header: "Process",
    accessor: "name",
    disableSortBy: true,
    Cell: (cellProps: ITableCellProps) => {
      const { name, type } = cellProps.row.original;
      if (type === "software_install") {
        return <SetupSoftwareProcessCell name={name || "Unknown software"} />;
      }
      if (type === "script_run" || type === "software_script_run") {
        return <SetupScriptProcessCell name={name || "Unknown script"} />;
      }
      return null;
    },
  },
  {
    Header: "Status",
    accessor: "status",
    disableSortBy: true,
    Cell: (cellProps: ITableCellProps) => {
      const { status, type } = cellProps.row.original;
      if (type === "software_install") {
        return <SetupSoftwareStatusCell status={status || "pending"} />;
      }
      if (type === "script_run" || type === "software_script_run") {
        return <SetupScriptStatusCell status={status || "pending"} />;
      }
      return null;
    },
  },
];

export default generateColumnConfigs;
