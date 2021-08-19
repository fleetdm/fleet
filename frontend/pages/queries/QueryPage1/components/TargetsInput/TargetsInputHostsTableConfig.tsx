import React from "react";
import { Cell, UseRowSelectInstanceProps } from "react-table";

import { IDataColumn } from "interfaces/datatable_config";

// @ts-ignore
import Checkbox from "components/forms/fields/Checkbox";
import TextCell from "components/TableContainer/DataTable/TextCell";
import StatusCell from "components/TableContainer/DataTable/StatusCell/StatusCell";

interface ITargetHostsTableData {
  hostname: string;
  status: string;
  primary_ip: string;
  primary_mac: string;
  os_version: string;
  osquery_version: string;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
export const generateTableHeaders = (shouldShowSelectionHeader: boolean): IDataColumn[] => {
  const selectionHeader = shouldShowSelectionHeader ? [
    {
      id: "selection",
      Header: (cellProps: UseRowSelectInstanceProps<ITargetHostsTableData>): JSX.Element => {
        const props = cellProps.getToggleAllRowsSelectedProps();
        const checkboxProps = {
          value: props.checked,
          indeterminate: props.indeterminate,
          onChange: () => cellProps.toggleAllRowsSelected(),
        };
        return <Checkbox {...checkboxProps} />;
      },
      Cell: (cellProps: Cell): JSX.Element => {
        const props = cellProps.row.getToggleRowSelectedProps();
        const checkboxProps = {
          value: props.checked,
          onChange: () => cellProps.row.toggleRowSelected(),
        };
        return <Checkbox {...checkboxProps} />;
      },
      disableHidden: true,
    },
  ] : [];
  
  return [
    ...selectionHeader,
    {
      title: "Hostname",
      Header: "Hostname",
      disableSortBy: true,
      accessor: "hostname",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      title: "Status",
      Header: "Status",
      disableSortBy: true,
      accessor: "status",
      Cell: (cellProps) => <StatusCell value={cellProps.cell.value} />,
    },
    {
      title: "IP address",
      Header: "IP address",
      accessor: "primary_ip",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      title: "MAC address",
      Header: "MAC address",
      accessor: "primary_mac",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      title: "OS",
      Header: "OS",
      accessor: "os_version",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      title: "Osquery",
      Header: "Osquery",
      accessor: "osquery_version",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
  ];
};

export default null;
