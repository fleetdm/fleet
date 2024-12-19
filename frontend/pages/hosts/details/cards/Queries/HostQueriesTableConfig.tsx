import React from "react";

import { IQueryStats } from "interfaces/query_stats";
import { getPerformanceImpactDescription } from "utilities/helpers";

import TextCell from "components/TableContainer/DataTable/TextCell";
import PerformanceImpactCell from "components/TableContainer/DataTable/PerformanceImpactCell";
import TooltipWrapper from "components/TooltipWrapper";
import ReportUpdatedCell from "pages/hosts/details/cards/Queries/ReportUpdatedCell";
import Icon from "components/Icon";

interface IHostQueriesTableData extends Partial<IQueryStats> {
  performance: { indicator: string; id: number };
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
      title: "Query",
      Header: "Query",
      disableSortBy: true,
      accessor: "query_name",
      Cell: (cellProps: ICellProps) => (
        <TextCell value={cellProps.cell.value} />
      ),
      sortType: "caseInsensitive",
    },
    {
      Header: () => {
        return (
          <TooltipWrapper
            tipContent={
              <>
                This is the performance <br />
                impact on this host.
              </>
            }
          >
            Performance impact
          </TooltipWrapper>
        );
      },
      disableSortBy: true,
      accessor: "performance",
      Cell: (cellProps: IPerformanceImpactCell) => {
        const baseClass = "performance-cell";
        return (
          <span className={baseClass}>
            <PerformanceImpactCell
              value={cellProps.cell.value}
              customIdPrefix="query-perf-pill"
              isHostSpecific
            />
            {!queryReportsDisabled &&
              cellProps.row.original.should_link_to_hqr && (
                <Icon
                  name="chevron-right"
                  className={`${baseClass}__link-icon`}
                  color="core-fleet-blue"
                />
              )}
          </span>
        );
      },
    },
  ];

  // include the Report updated column if query reports are globally enabled
  if (!queryReportsDisabled) {
    cols.push({
      Header: "Report updated",
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
      user_time,
      system_time,
      executions,
      query_name,
      scheduled_query_id,
      last_fetched,
      interval,
      discard_data,
      automations_enabled,
    } = query;
    // getPerformanceImpactDescription takes aggregate p50 values
    // getPerformanceImpactDescription takes aggregate p50 values so we need to divide by total executions in order to show average performance per query execution
    const scheduledQueryPerformance = {
      user_time_p50: executions > 0 ? user_time / executions : 0,
      system_time_p50: executions > 0 ? system_time / executions : 0,
      total_executions: executions,
    };
    return {
      query_name,
      id: scheduled_query_id,
      performance: {
        indicator: getPerformanceImpactDescription(scheduledQueryPerformance),
        id: scheduled_query_id,
      },
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
