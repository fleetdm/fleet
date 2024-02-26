import React, { useCallback, useMemo } from "react";

import { IQueryStats } from "interfaces/query_stats";
import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";
import CustomLink from "components/CustomLink";
import PATHS from "router/paths";
import { InjectedRouter } from "react-router";
import { Row } from "react-table";

import {
  generateColumnConfigs,
  generateDataSet,
} from "./HostQueriesTableConfig";

const baseClass = "host-queries";

interface IHostQueriesProps {
  hostId: number;
  schedule?: IQueryStats[];
  isChromeOSHost: boolean;
  queryReportsDisabled?: boolean;
  router: InjectedRouter;
}

interface IHostQueriesRowProps extends Row {
  original: {
    id?: number;
    should_link_to_hqr?: boolean;
  };
}
const HostQueries = ({
  hostId,
  schedule,
  isChromeOSHost,
  queryReportsDisabled,
  router,
}: IHostQueriesProps): JSX.Element => {
  const renderEmptyQueriesTab = () => {
    if (isChromeOSHost) {
      return (
        <EmptyTable
          header="Scheduled queries are not supported for this host"
          info={
            <>
              <span>Interested in collecting data from your Chromebooks? </span>
              <CustomLink
                url="https://www.fleetdm.com/contact"
                text="Let us know"
                newTab
              />
            </>
          }
        />
      );
    }
    return (
      <EmptyTable
        header="No queries are scheduled to run on this host"
        info={
          <>
            Expecting to see queries? Try selecting <b>Refetch</b> to ask this
            host to report fresh vitals.
          </>
        }
      />
    );
  };

  const onSelectSingleRow = useCallback(
    (row: IHostQueriesRowProps) => {
      const { id: queryId, should_link_to_hqr } = row.original;

      if (!hostId || !queryId || !should_link_to_hqr || queryReportsDisabled) {
        return;
      }
      router.push(`${PATHS.HOST_QUERY_REPORT(hostId, queryId)}`);
    },
    [hostId, queryReportsDisabled, router]
  );

  const tableData = useMemo(() => generateDataSet(schedule ?? []), [schedule]);

  const columnConfigs = useMemo(
    () => generateColumnConfigs(queryReportsDisabled),
    [queryReportsDisabled]
  );

  return (
    <div className={`section section--${baseClass}`}>
      <p className="section__header">Queries</p>
      {!schedule || !schedule.length || isChromeOSHost ? (
        renderEmptyQueriesTab()
      ) : (
        <div>
          <TableContainer
            columnConfigs={columnConfigs}
            data={tableData}
            onQueryChange={() => null}
            resultsTitle="queries"
            defaultSortHeader="query_name"
            defaultSortDirection="asc"
            showMarkAllPages={false}
            isAllPagesSelected={false}
            emptyComponent={() => <></>}
            disablePagination
            disableCount
            disableMultiRowSelect
            isLoading={false} // loading state handled at parent level
            onSelectSingleRow={onSelectSingleRow}
          />
        </div>
      )}
    </div>
  );
};

export default HostQueries;
