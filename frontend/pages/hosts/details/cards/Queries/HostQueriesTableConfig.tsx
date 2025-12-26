import React from "react";

import { IQueryStats } from "interfaces/query_stats";

import TooltipTruncatedTextCell from "components/TableContainer/DataTable/TooltipTruncatedTextCell";
import TooltipWrapper from "components/TooltipWrapper";
import ReportUpdatedCell from "pages/hosts/details/cards/Queries/ReportUpdatedCell";

interface IHostQueriesTableData extends Partial<IQueryStats> {
  should_link_to_hqr: boolean;
  id: number;
}
interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

interface IRowProps {
  row: {
    original: IHostQueriesTableData;
  };
}

interface ICellProps extends IRowProps {
  cell: {
    value: string | number | boolean;
  };
}

interface IPerformanceImpactCell extends IRowProps {
  cell: {
    value: {
      indicator: string;
      id: number;
    };
  };
}

interface IDataColumn {
  title?: string;
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  accessor: string;
  Cell:
    | ((props: ICellProps) => JSX.Element)
    | ((props: IPerformanceImpactCell) => JSX.Element);
  disableHidden?: boolean;
  disableSortBy?: boolean;
  sortType?: string;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateColumnConfigs = (
  hostId: number,
  queryReportsDisabled?: boolean
): IDataColumn[] => {
  const cols: IDataColumn[] = [
    {
      Header: "Name",
      disableSortBy: true,
      accessor: "query_name",
      Cell: (cellProps: ICellProps) => (
        <TooltipTruncatedTextCell value={cellProps.cell.value} />
      ),
      sortType: "caseInsensitive",
    },
  ];

  // include the Report updated column if query reports are globally enabled
  if (!queryReportsDisabled) {
    cols.push({
      Header: () => {
        return (
          <TooltipWrapper
            tipContent={
              <>
                Each query is updated based on an <br />
                individually set interval.
              </>
            }
          >
            Last updated
          </TooltipWrapper>
        );
      },
      disableSortBy: true,
      accessor: "last_fetched", // tbd - may change
      Cell: (cellProps: ICellProps) => {
        return (
          <ReportUpdatedCell
            {...cellProps.row.original}
            hostId={hostId}
            queryId={cellProps.row.original.id}
          />
        );
      },
    });
  }
  return cols;
};

const enhanceScheduleData = (
  query_stats: IQueryStats[]
): IHostQueriesTableData[] => {
  return Object.values(query_stats).map((query) => {
    const {
      query_name,
      scheduled_query_id,
      last_fetched,
      interval,
      discard_data,
      automations_enabled,
    } = query;
    return {
      query_name,
      id: scheduled_query_id,
      last_fetched,
      interval,
      discard_data,
      automations_enabled,
      should_link_to_hqr: !!last_fetched || (!!interval && !discard_data),
    };
  });
};

const generateDataSet = (
  query_stats: IQueryStats[]
): IHostQueriesTableData[] => {
  return query_stats ? enhanceScheduleData(query_stats) : [];
};

export { generateColumnConfigs, generateDataSet };
