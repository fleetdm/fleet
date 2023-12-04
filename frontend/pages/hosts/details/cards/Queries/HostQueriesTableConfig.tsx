import React from "react";

import { IQueryStats } from "interfaces/query_stats";
import { performanceIndicator } from "utilities/helpers";

import TextCell from "components/TableContainer/DataTable/TextCell";
import PillCell from "components/TableContainer/DataTable/PillCell";
import TooltipWrapper from "components/TooltipWrapper";
import { HumanTimeDiffWithFleetLaunchCutoff } from "components/HumanTimeDiffWithDateTip";
import { uniqueId } from "lodash";
import ReactTooltip from "react-tooltip";
import { COLORS } from "styles/var/colors";

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

interface IHostQueriesTable extends Partial<IQueryStats> {
  performance: { indicator: string; id: number };
  should_link_to_hqr: boolean;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateColumnConfigs = (
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

  // include the Report updated column if query reports are globally enabled
  if (!queryReportsDisabled) {
    cols.push({
      Header: "Report updated",
      disableSortBy: true,
      accessor: "last_fetched", // tbd - may change
      Cell: (cellProps: ICellProps) => {
        const {
          last_fetched,
          interval,
          discard_data,
          automations_enabled,
        } = cellProps.row.original;

        // if this query doesn't have an interval, it either has a stored report from previous runs
        // and will link to that report, or won't be included in this data in the first place.
        if (interval) {
          if (discard_data && automations_enabled) {
            // TODO: this is the only case where the row is NOT clickable with a link to the host's HQR
            // query runs, sends results to a logging dest, doesn't cache
            return (
              <TextCell
                greyed
                emptyCellTooltipText={
                  <>
                    Results from this query are not reported in Fleet.
                    <br />
                    Data is being sent to your log destination.
                  </>
                }
              />
            );
          }

          // Query is scheduled to run on host, but hasn't yet
          if (!last_fetched) {
            const tipId = uniqueId();
            return (
              <TextCell
                value="Never"
                formatter={(val) => (
                  <>
                    <span data-tip data-for={tipId}>
                      {val}
                    </span>
                    <ReactTooltip
                      id={tipId}
                      effect="solid"
                      backgroundColor={COLORS["tooltip-bg"]}
                      place="top"
                    >
                      This query has not run on this host.
                    </ReactTooltip>
                  </>
                )}
                greyed
              />
            );
          }
        }

        // render with link to cached results
        return (
          <TextCell
            // last_fetched will be truthy at this point
            value={{ timeString: last_fetched ?? "" }}
            formatter={HumanTimeDiffWithFleetLaunchCutoff}
          />
        );
      },
    });
  }
  return cols;
};

const enhanceScheduleData = (
  query_stats: IQueryStats[]
): IHostQueriesTable[] => {
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
    const scheduledQueryPerformance = {
      user_time_p50: user_time,
      system_time_p50: system_time,
      total_executions: executions,
    };
    return {
      query_name,
      id: scheduled_query_id,
      performance: {
        indicator: performanceIndicator(scheduledQueryPerformance),
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

const generateDataSet = (query_stats: IQueryStats[]): IHostQueriesTable[] => {
  return query_stats ? [...enhanceScheduleData(query_stats)] : [];
};

export { generateColumnConfigs, generateDataSet };
