import React from "react";
import { capitalize } from "lodash";
import { Link } from "react-router";
import PATHS from "router/paths";

import { IMunkiIssuesAggregate } from "interfaces/macadmins";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import TruncatedTextCell from "components/TableContainer/DataTable/TruncatedTextCell";
import TooltipWrapper from "components/TooltipWrapper";

import Chevron from "../../../../../assets/images/icon-chevron-right-9x6@2x.png";

const TAGGED_TEMPLATES = {
  hostsByMunkiIssue: (munkiIssueId: number) => {
    return `?munki_issue_id=${munkiIssueId}`;
  },
};

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
interface ICellProps {
  cell: {
    value: string;
  };
  row: {
    original: IMunkiIssuesAggregate;
  };
}

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
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

const munkiIssuesTableHeaders = [
  {
    title: "Issue",
    Header: (): JSX.Element => {
      const titleWithToolTip = (
        <TooltipWrapper
          tipContent={`
            Issues reported the last time Munki ran on each host.
          `}
        >
          Issue
        </TooltipWrapper>
      );
      return <HeaderCell value={titleWithToolTip} disableSortBy />;
    },
    disableSortBy: true,
    accessor: "name",
    Cell: (cellProps: ICellProps) => (
      <TruncatedTextCell value={cellProps.cell.value} />
    ),
  },
  {
    title: "Type",
    Header: "Type",
    disableSortBy: true,
    accessor: "type",
    Cell: (cellProps: ICellProps) => (
      <TextCell value={capitalize(cellProps.cell.value)} />
    ),
  },
  {
    title: "Hosts",
    Header: (headerProps: IHeaderProps): JSX.Element => {
      return (
        <HeaderCell
          value={"Hosts"}
          isSortedDesc={headerProps.column.isSortedDesc}
        />
      );
    },
    disableSortBy: false,
    accessor: "hosts_count",
    Cell: (cellProps: ICellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "",
    Header: "",
    accessor: "linkToFilteredHosts",
    disableSortBy: true,
    Cell: (cellProps: ICellProps) => {
      return (
        <>
          {cellProps.row.original && (
            <Link
              to={
                PATHS.MANAGE_HOSTS +
                TAGGED_TEMPLATES.hostsByMunkiIssue(cellProps.row.original.id)
              }
              className={`issue-link`}
            >
              View all hosts{" "}
              <img alt="link to hosts filtered by policy ID" src={Chevron} />
            </Link>
          )}
        </>
      );
    },
  },
];

export default munkiIssuesTableHeaders;
