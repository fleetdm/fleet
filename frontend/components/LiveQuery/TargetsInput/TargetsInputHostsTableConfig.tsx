/* eslint-disable react/prop-types */

import React from "react";
import { Column, Row } from "react-table";

import { IStringCellProps } from "interfaces/datatable_config";
import { IHost } from "interfaces/host";

import TextCell from "components/TableContainer/DataTable/TextCell";
import LiveQueryIssueCell from "components/TableContainer/DataTable/LiveQueryIssueCell/LiveQueryIssueCell";
import StatusIndicator from "components/StatusIndicator";
import Icon from "components/Icon/Icon";

export type ITargestInputHostTableConfig = Column<IHost>;
type ITableStringCellProps = IStringCellProps<IHost>;

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
export const generateTableHeaders = (
  handleRowRemove?: (value: Row<IHost>) => void
): ITargestInputHostTableConfig[] => {
  const deleteHeader = handleRowRemove
    ? [
        {
          id: "delete",
          Header: "",
          Cell: (cellProps: ITableStringCellProps) => (
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
      Header: "Host",
      accessor: "display_name",
      Cell: (cellProps: ITableStringCellProps) => {
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
      Header: "Status",
      disableSortBy: true,
      accessor: "status",
      Cell: (cellProps) => <StatusIndicator value={cellProps.cell.value} />,
    },
    {
      Header: "Private IP address",
      accessor: "primary_ip",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      Header: "MAC address",
      accessor: "primary_mac",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      Header: "OS",
      accessor: "os_version",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      Header: "Osquery",
      accessor: "osquery_version",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    ...deleteHeader,
  ];
};

export default null;
