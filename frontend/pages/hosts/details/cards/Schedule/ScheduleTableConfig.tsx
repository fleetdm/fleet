import React from "react";

import { IQueryStats } from "interfaces/query_stats";
import { performanceIndicator, secondsToDhms } from "utilities/helpers";

import TextCell from "components/TableContainer/DataTable/TextCell";
import PillCell from "components/TableContainer/DataTable/PillCell";
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

interface IPillCellProps extends IRowProps {
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
    | ((props: IPillCellProps) => JSX.Element);
  disableHidden?: boolean;
  disableSortBy?: boolean;
}

interface IScheduleTable extends Partial<IQueryStats> {
  frequency: string;
  performance: { indicator: string; id: number };
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateTableHeaders = (): IDataColumn[] => {
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
      Cell: (cellProps: IPillCellProps) => (
        <PillCell
          value={cellProps.cell.value}
          customIdPrefix="query-perf-pill"
          hostDetails
        />
      ),
    },
  ];
};

const enhanceScheduleData = (query_stats: IQueryStats[]): IScheduleTable[] => {
  return Object.values(query_stats).map((query) => {
    const scheduledQueryPerformance = {
      user_time_p50: query.user_time,
      system_time_p50: query.system_time,
      total_executions: query.executions,
    };
    return {
      query_name: query.query_name,
      frequency: secondsToDhms(query.interval),
      performance: {
        indicator: performanceIndicator(scheduledQueryPerformance),
        id: query.scheduled_query_id,
      },
    };
  });
};

const generateDataSet = (query_stats: IQueryStats[]): IScheduleTable[] => {
  if (!query_stats) {
    return query_stats;
  }

  return [...enhanceScheduleData(query_stats)];
};

export { generateTableHeaders, generateDataSet };
