import React from "react";
import { uniqueId } from "lodash";

import TextCell from "components/TableContainer/DataTable/TextCell";
import PillCell from "components/TableContainer/DataTable/PillCell";
import { IQueryStats } from "interfaces/query_stats";
import {
  humanQueryLastRun,
  performanceIndicator,
  secondsToHms,
} from "fleet/helpers";
import IconToolTip from "components/IconToolTip";

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
    original: IQueryStats;
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

interface IPackTable extends IQueryStats {
  frequency: string;
  last_run: string;
  performance: (string | number)[];
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generatePackTableHeaders = (): IDataColumn[] => {
  return [
    {
      title: "Query name",
      Header: "Query name",
      disableSortBy: true,
      accessor: "query_name",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      title: "Frequency",
      Header: "Frequency",
      disableSortBy: true,
      accessor: "frequency",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      title: "Last run",
      Header: () => {
        return (
          <>
            Last run
            <IconToolTip
              isHtml
              text={`The last time the query ran<br/>since the last time osquery <br/>started on this host.`}
            />
          </>
        );
      },
      disableSortBy: true,
      accessor: "last_run",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      title: "Performance impact",
      Header: "Performance impact",
      disableSortBy: true,
      accessor: "performance",
      Cell: (cellProps) => (
        <PillCell
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
    };
    return {
      scheduled_query_name: query.scheduled_query_name,
      scheduled_query_id: query.scheduled_query_id,
      query_name: query.query_name,
      pack_name: query.pack_name,
      pack_id: query.pack_id,
      description: query.description,
      interval: query.interval,
      last_executed: query.last_executed,
      frequency: secondsToHms(query.interval),
      last_run: humanQueryLastRun(query.last_executed),
      performance: [
        performanceIndicator(scheduledQueryPerformance),
        query.scheduled_query_id || uniqueId(),
      ],
      average_memory: query.average_memory,
      denylisted: query.denylisted,
      executions: query.executions,
      system_time: query.system_time,
      user_time: query.user_time,
    };
  });
};

const generatePackDataSet = (query_stats: IQueryStats[]): IPackTable[] => {
  // Cannot pass undefined to enhancePackData
  if (!query_stats) {
    return query_stats;
  }

  return [...enhancePackData(query_stats)];
};

export { generatePackTableHeaders, generatePackDataSet };
