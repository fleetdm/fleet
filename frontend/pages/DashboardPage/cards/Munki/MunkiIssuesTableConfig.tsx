import React from "react";
import { capitalize } from "lodash";

import { IMunkiIssuesAggregate } from "interfaces/macadmins";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import TruncatedTextCell from "components/TableContainer/DataTable/TruncatedTextCell";
import TooltipWrapper from "components/TooltipWrapper";
import ViewAllHostsLink from "components/ViewAllHostsLink";

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
            <ViewAllHostsLink
              queryParams={{ munki_issue_id: cellProps.row.original.id }}
              className="munki-issue-link"
            />
          )}
        </>
      );
    },
  },
];

export default munkiIssuesTableHeaders;
