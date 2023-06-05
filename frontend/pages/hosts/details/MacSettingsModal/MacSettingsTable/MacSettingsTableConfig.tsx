import TextCell from "components/TableContainer/DataTable/TextCell";
import React from "react";
import { IHostMacMdmProfile } from "interfaces/mdm";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import TruncatedTextCell from "components/TableContainer/DataTable/TruncatedTextCell";
import MacSettingStatusCell from "./MacSettingStatusCell";

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

interface ICellProps {
  cell: {
    value: string;
  };
  row: {
    original: IHostMacMdmProfile;
  };
}

interface IDataColumn {
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  Cell: (props: ICellProps) => JSX.Element;
  id?: string;
  title?: string;
  accessor?: string;
  disableHidden?: boolean;
  disableSortBy?: boolean;
  sortType?: string;
}

const tableHeaders: IDataColumn[] = [
  {
    title: "Name",
    Header: "Name",
    disableSortBy: true,
    accessor: "name",
    Cell: (cellProps: ICellProps): JSX.Element => (
      <TextCell value={cellProps.cell.value} />
    ),
  },
  {
    title: "Status",
    Header: "Status",
    disableSortBy: true,
    accessor: "statusText",
    Cell: (cellProps: ICellProps) => {
      return (
        <MacSettingStatusCell
          name={cellProps.row.original.name}
          status={cellProps.row.original.status}
          operationType={cellProps.row.original.operation_type}
        />
      );
    },
  },
  {
    title: "Error",
    Header: "Error",
    disableSortBy: true,
    accessor: "detail",
    Cell: (cellProps: ICellProps): JSX.Element => {
      const profile = cellProps.row.original;
      return (
        <TruncatedTextCell
          value={
            (profile.status === "failed" && profile.detail) ||
            DEFAULT_EMPTY_CELL_VALUE
          }
        />
      );
    },
  },
];

export default tableHeaders;
