import React from "react";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import TooltipWrapper from "components/TooltipWrapper";

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
    original: { user: string };
  };
}

interface IDataColumn {
  title?: string;
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  accessor: string;
  Cell: (props: ICellProps) => JSX.Element;
  disableHidden?: boolean;
  disableSortBy?: boolean;
  sortType?: string;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateTableHeaders = (): IDataColumn[] => {
  return [
    {
      title: "Username",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      disableSortBy: false,
      sortType: "caseInsensitive",
      accessor: "username",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      Header: () => {
        return (
          <TooltipWrapper
            tipContent={
              <>
                The command line shell, such as bash,
                <br />
                that this user is equipped with by
                <br />
                default when they log in to the system.
              </>
            }
          >
            Shell
          </TooltipWrapper>
        );
      },
      disableSortBy: true,
      accessor: "shell",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
  ];
};

export default generateTableHeaders;
