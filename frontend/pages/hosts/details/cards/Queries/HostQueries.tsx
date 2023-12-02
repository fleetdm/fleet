import React, { useCallback } from "react";

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
  hostId?: number;
  schedule?: IQueryStats[];
  isChromeOSHost: boolean;
  isLoading: boolean;
  queryReportsDisabled?: boolean;
  router: InjectedRouter;
}

interface IHostQueriesRowProps extends Row {
  original: {
    id?: number;
  };
}
const HostQueries = ({
  hostId,
  schedule,
  isChromeOSHost,
  isLoading,
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
      if (!hostId || !row.original.id) {
        return;
      }
      router.push(`${PATHS.HOST_QUERY_REPORT(hostId, row.original.id)}`);
    },
    [hostId, router]
  );

  return (
    <div className="section section--host-queries">
      <p className="section__header">Queries</p>
      {!schedule || !schedule.length || isChromeOSHost ? (
        renderEmptyQueriesTab()
      ) : (
        <div>
          <TableContainer
            columns={generateColumnConfigs(queryReportsDisabled)}
            data={generateDataSet(schedule)}
            onQueryChange={() => null}
            resultsTitle="queries"
            defaultSortHeader="scheduled_query_name"
            defaultSortDirection="asc"
            showMarkAllPages={false}
            isAllPagesSelected={false}
            emptyComponent={() => <></>}
            disablePagination
            disableCount
            disableMultiRowSelect
            {...{ isLoading, onSelectSingleRow }}
          />
        </div>
      )}
    </div>
  );
};

export default HostQueries;
