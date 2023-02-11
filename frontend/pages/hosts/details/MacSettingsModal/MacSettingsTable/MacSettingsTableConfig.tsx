import LinkCell from "components/TableContainer/DataTable/LinkCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import React from "react";
import { IMacSetting, IMacSettings } from "interfaces/mdm";

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
    original: IMacSetting;
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

const generateTableHeaders = (): IDataColumn[] => {
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
      accessor: "status",
      Cell: (cellProps: ICellProps): JSX.Element => {
        // TODO - logically generate mac setting status.
        // define new component, like StatusIndicator but with more options and icons ?
        return <div>Status of this mac setting</div>;
      },
    },
    {
      title: "Error",
      Header: "Error",
      disableSortBy: true,
      accessor: "error",
      Cell: (cellProps: ICellProps): JSX.Element => {
        // TODO: logically generate settings error
        return <div>Error</div>;
      },
    },
  ];

  return tableHeaders;
};

const generateDataSet = (hostMacSettings: IMacSettings) => {
  // TODO - make this real
  return [
    {
      name: "setting name",
      status: "setting status",
      error: "setting error",
    },
  ];
};

export { generateTableHeaders, generateDataSet };
