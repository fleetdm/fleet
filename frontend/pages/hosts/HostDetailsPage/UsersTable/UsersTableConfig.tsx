import React from "react";
import ReactTooltip from "react-tooltip";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import QuestionIcon from "../../../../../assets/images/icon-question-16x16@2x.png";

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}
interface ICellProps {
  cell: {
    value: any;
  };
  row: {
    original: { user: string };
  };
}

interface IDataColumn {
  title: string;
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  accessor: string;
  Cell: (props: ICellProps) => JSX.Element;
  disableHidden?: boolean;
  disableSortBy?: boolean;
  sortType?: string;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateUsersTableHeaders = (): IDataColumn[] => {
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
      title: "Shell",
      Header: () => {
        return (
          <div>
            <span>Shell</span>
            <span
              data-tip
              data-for="host-users-table__shell-tooltip"
              data-tip-disable={false}
            >
              <img alt="question icon" src={QuestionIcon} />
            </span>
            <ReactTooltip
              className="host-users-table__shell-tooltip"
              place="bottom"
              type="dark"
              effect="solid"
              backgroundColor="#3e4771"
              id="host-users-table__shell-tooltip"
              data-html
            >
              <div style={{ textAlign: "center" }}>
                The command line shell, such as bash,
                <br />
                that this user is equipped with by default
                <br />
                when they log in to the system.
              </div>
            </ReactTooltip>
          </div>
        );
      },
      disableSortBy: true,
      accessor: "shell",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
  ];
};

export default generateUsersTableHeaders;
