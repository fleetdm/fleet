/* eslint-disable react/prop-types */

import React from "react";
import { Row } from "react-table";

import { IDataColumn } from "interfaces/datatable_config";

import TextCell from "components/TableContainer/DataTable/TextCell";
import StatusCell from "components/TableContainer/DataTable/StatusCell/StatusCell";
import RemoveIcon from "../../../../assets/images/icon-action-remove-20x20@2x.png";

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
export const generateTableHeaders = (
  handleRowRemove?: (value: Row) => void
): IDataColumn[] => {
  const deleteHeader = handleRowRemove
    ? [
        {
          id: "delete",
          Header: "",
          Cell: (cellProps: { row: Row }): JSX.Element => (
            <div onClick={() => handleRowRemove(cellProps.row)}>
              <img alt="Remove" src={RemoveIcon} />
            </div>
          ),
          disableHidden: true,
        },
      ]
    : [];

  return [
    {
      title: "Host",
      Header: "Host",
      disableSortBy: true,
      accessor: "display_name",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    // TODO: Consider removing status column from selected hosts table because
    // status info is not refreshed once a target has been selected
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
