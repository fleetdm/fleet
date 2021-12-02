/* eslint-disable react/prop-types */

import React from "react";
import { Cell } from "react-table";

import { IDataColumn } from "interfaces/datatable_config";

// @ts-ignore
import TextCell from "components/TableContainer/DataTable/TextCell";
import StatusCell from "components/TableContainer/DataTable/StatusCell/StatusCell";
import RemoveIcon from "../../../assets/images/icon-action-remove-20x20@2x.png";

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
export const generateTableHeaders = (showDelete: boolean): IDataColumn[] => {
  const deleteHeader = showDelete
    ? [
        {
          id: "delete",
          Header: "",
          Cell: (cellProps: Cell): JSX.Element => (
            <div>
              <img alt="Remove" src={RemoveIcon} />
            </div>
          ),
          disableHidden: true,
        },
      ]
    : [];

  return [
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
    ...deleteHeader,
  ];
};

export default null;
