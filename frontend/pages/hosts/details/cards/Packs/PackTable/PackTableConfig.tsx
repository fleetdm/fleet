import React from "react";
import { uniqueId } from "lodash";

import { IQueryStats } from "interfaces/query_stats";
import {
  humanQueryLastRun,
  getPerformanceImpactDescription,
  secondsToHms,
} from "utilities/helpers";

import TextCell from "components/TableContainer/DataTable/TextCell";
import PerformanceImpactCell from "components/TableContainer/DataTable/PerformanceImpactCell";
import TooltipWrapper from "components/TooltipWrapper";

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

interface IRowProps {
  row: {
    original: IQueryStats;
  };
}

interface ICellProps extends IRowProps {
  cell: {
    value: string | number | boolean;
  };
}

interface IPerformanceImpactCell extends IRowProps {
  cell: {
    value: { indicator: string; id: number };
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
}

interface IPackTable extends Partial<IQueryStats> {
  frequency: string;
  last_run: string;
  performance: { indicator: string; id: number };
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generatePackTableHeaders = (): IDataColumn[] => {
  return [
    {
      title: "Query",
      Header: "Query",
      disableSortBy: true,
      accessor: "query_name",
      Cell: (cellProps: ICellProps) => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      title: "Frequency",
      Header: "Frequency",
      disableSortBy: true,
      accessor: "frequency",
      Cell: (cellProps: ICellProps) => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      Header: () => {
        return (
          <TooltipWrapper
            tipContent={
              <>
                The last time the query ran
                <br />
                since the last time osquery <br />
                started on this host.
              </>
            }
          >
            Last run
          </TooltipWrapper>
        );
      },
      disableSortBy: true,
      accessor: "last_run",
      Cell: (cellProps: ICellProps) => (
        <TextCell value={cellProps.cell.value} />
      ),
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
      Cell: (cellProps: IPerformanceImpactCell) => (
        <PerformanceImpactCell
          value={cellProps.cell.value}
          customIdPrefix="query-perf-pill"
        />
      ),
    },
  ];
};

const enhancePackData = (query_stats: IQueryStats[]): IPackTable[] => {
  return Object.values(query_stats).map((query) => {
    const scheduledQueryPerformance = {
      user_time_p50: query.user_time,
      system_time_p50: query.system_time,
      total_executions: query.executions,
      query_denylisted: query.denylisted,
    };
    return {
      query_name: query.query_name,
      last_executed: query.last_executed,
      frequency: secondsToHms(query.interval),
      last_run: humanQueryLastRun(query.last_executed),
      performance: {
        indicator: getPerformanceImpactDescription(scheduledQueryPerformance),
        id: query.scheduled_query_id || parseInt(uniqueId(), 10),
      },
    };
  });
};

const generatePackDataSet = (query_stats: IQueryStats[]): IPackTable[] => {
  if (!query_stats) {
    return query_stats;
  }

  return [...enhancePackData(query_stats)];
};

export { generatePackTableHeaders, generatePackDataSet };
