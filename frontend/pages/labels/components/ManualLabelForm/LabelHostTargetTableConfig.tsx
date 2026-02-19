/* eslint-disable react/prop-types */

import React from "react";
import { Column, Row } from "react-table";

import { IStringCellProps } from "interfaces/datatable_config";
import { IHost } from "interfaces/host";

import TextCell from "components/TableContainer/DataTable/TextCell";
import Icon from "components/Icon/Icon";
import Button from "components/buttons/Button";

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
            <Button
              onClick={() => handleRowRemove(cellProps.row)}
              variant="icon"
            >
              <Icon name="close-filled" />
            </Button>
          ),
          disableHidden: true,
        },
      ]
    : [];

  return [
    {
      Header: "Host",
      accessor: "display_name",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      Header: "Hostname",
      accessor: "hostname",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      Header: "Serial number",
      accessor: "hardware_serial",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    ...deleteHeader,
  ];
};

export default null;
