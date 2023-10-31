/* eslint-disable react/prop-types */

import React from "react";
import { Row } from "react-table";

import { IDataColumn } from "interfaces/datatable_config";
import { IHost } from "interfaces/host";

import TextCell from "components/TableContainer/DataTable/TextCell";
import LiveQueryIssueCell from "components/TableContainer/DataTable/LiveQueryIssueCell/LiveQueryIssueCell";
import StatusIndicator from "components/StatusIndicator";
import Icon from "components/Icon/Icon";

interface ICellProps {
  cell: {
    value: string;
  };
  row: {
    original: IHost;
  };
}

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
              <Icon name="close-filled" />
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
      accessor: "display_name",
      Cell: (cellProps: ICellProps) => {
        return (
          <LiveQueryIssueCell
            displayName={cellProps.cell.value}
            distributedInterval={cellProps.row.original.distributed_interval}
            status={cellProps.row.original.status}
            rowId={cellProps.row.original.id}
          />
        );
      },
    },
    // TODO: Consider removing status column from selected hosts table because
    // status info is not refreshed once a target has been selected
    {
      title: "Status",
      Header: "Status",
      disableSortBy: true,
      accessor: "status",
      Cell: (cellProps) => <StatusIndicator value={cellProps.cell.value} />,
    },
    {
      title: "Private IP address",
      Header: "Private IP address",
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
