import React from "react";
import { Link } from "react-router";
import ReactTooltip from "react-tooltip";
import { isEmpty } from "lodash";

import PATHS from "router/paths";

import { ISoftware } from "interfaces/software";

import TextCell from "components/TableContainer/DataTable/TextCell";
import Chevron from "../../../../../assets/images/icon-chevron-blue-16x16@2x.png";
import IssueIcon from "../../../../../assets/images/icon-issue-fleet-black-50-16x16@2x.png";

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
    original: ISoftware;
  };
}

interface IDataColumn {
  title: string;
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  accessor: string;
  Cell: (props: ICellProps) => JSX.Element;
  disableHidden?: boolean;
  disableSortBy?: boolean;
}

const vulnerabilityTableHeader = [
  {
    title: "Vulnerabilities",
    Header: "",
    disableSortBy: true,
    accessor: "vulnerabilities",
    Cell: (cellProps: ICellProps) => {
      const vulnerabilities = cellProps.cell.value;
      if (isEmpty(vulnerabilities)) {
        return <></>;
      }
      return (
        <>
          <span
            className={`vulnerabilities tooltip__tooltip-icon`}
            data-tip
            data-for={`vulnerabilities__${cellProps.row.original.id.toString()}`}
            data-tip-disable={false}
          >
            <img alt="software vulnerabilities" src={IssueIcon} />
          </span>
          <ReactTooltip
            place="bottom"
            type="dark"
            effect="solid"
            backgroundColor="#3e4771"
            id={`vulnerabilities__${cellProps.row.original.id.toString()}`}
            data-html
          >
            <span className={`vulnerabilities tooltip__tooltip-text`}>
              {vulnerabilities.length === 1
                ? "1 vulnerability detected"
                : `${vulnerabilities.length} vulnerabilities detected`}
            </span>
          </ReactTooltip>
        </>
      );
    },
  },
];

const softwareTableHeaders = [
  {
    title: "Name",
    Header: "Name",
    disableSortBy: true,
    accessor: "name",
    Cell: (cellProps: ICellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Version",
    Header: "Version",
    disableSortBy: true,
    accessor: "version",
    Cell: (cellProps: ICellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Actions",
    Header: "",
    disableSortBy: true,
    accessor: "id",
    Cell: (cellProps: ICellProps) => {
      return (
        <Link
          to={`${PATHS.MANAGE_HOSTS}?software_id=${cellProps.cell.value}`}
          className="software-link"
        >
          <img alt="link to hosts filtered by software ID" src={Chevron} />
        </Link>
      );
    },
  },
];

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateTableHeaders = (): IDataColumn[] => {
  return softwareTableHeaders;
};

const generateModalSoftwareTableHeaders = (): IDataColumn[] => {
  return vulnerabilityTableHeader.concat(softwareTableHeaders);
};

export { generateTableHeaders, generateModalSoftwareTableHeaders };
