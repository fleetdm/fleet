import React from "react";

import { IMdmSolution } from "interfaces/mdm";

import TextCell from "components/TableContainer/DataTable/TextCell";
import ViewAllHostsLink from "components/ViewAllHostsLink";
import TooltipWrapper from "components/TooltipWrapper";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties

interface IMDMSolutionWithPlatformId extends IMdmSolution {
  selectedPlatformLabelId?: number;
}

interface ICellProps {
  cell: {
    value: string;
  };
  row: {
    original: IMDMSolutionWithPlatformId;
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
  disableGlobalFilter?: boolean;
  disableHidden?: boolean;
  disableSortBy?: boolean;
}

export const generateSolutionsTableHeaders = (
  teamId?: number
): IDataColumn[] => [
  {
    title: "Server URL",
    Header: (): JSX.Element => {
      const titleWithToolTip = (
        <TooltipWrapper
          position="top-start"
          tipContent={
            <>
              The MDM server URL is used to connect hosts with the MDM service.
              For cross-platform MDM solutions, each operating system has a
              different URL.
            </>
          }
          className="server-url-header"
        >
          Status
        </TooltipWrapper>
      );
      return <HeaderCell value={titleWithToolTip} disableSortBy />;
    },
    disableSortBy: true,
    accessor: "server_url",
    Cell: (cellProps: ICellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Hosts",
    Header: "Hosts",
    disableSortBy: true,
    accessor: "hosts_count",
    Cell: (cellProps: ICellProps) => (
      <div className="host-count-cell">
        <TextCell value={cellProps.cell.value} classes="" />
        <ViewAllHostsLink
          queryParams={{ mdm_id: cellProps.row.original.id, team_id: teamId }}
          className="view-mdm-solution-link"
          platformLabelId={cellProps.row.original.selectedPlatformLabelId}
          rowHover
        />
      </div>
    ),
  },
];

const enhanceSolutionsData = (
  solutions: IMdmSolution[],
  selectedPlatformLabelId?: number
): IMdmSolution[] => {
  return Object.values(solutions).map((solution) => {
    return {
      id: solution.id,
      name: solution.name || "Unknown",
      server_url: solution.server_url,
      hosts_count: solution.hosts_count,
      selectedPlatformLabelId,
    };
  });
};

export const generateSolutionsDataSet = (
  solutions: IMdmSolution[] | null,
  selectedPlatformLabelId?: number
): IMdmSolution[] => {
  if (!solutions) {
    return [];
  }
  return [...enhanceSolutionsData(solutions, selectedPlatformLabelId)];
};
