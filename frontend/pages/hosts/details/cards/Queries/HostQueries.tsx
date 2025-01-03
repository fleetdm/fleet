import React, { useCallback, useMemo } from "react";

import { IQueryStats } from "interfaces/query_stats";
import { SUPPORT_LINK } from "utilities/constants";
import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";
import CustomLink from "components/CustomLink";
import Card from "components/Card";
import PATHS from "router/paths";
import { InjectedRouter } from "react-router";
import { Row } from "react-table";

import {
  generateColumnConfigs,
  generateDataSet,
} from "./HostQueriesTableConfig";

const baseClass = "host-queries-card";

interface IHostQueriesProps {
  hostId: number;
  schedule?: IQueryStats[];
  hostPlatform: string;
  queryReportsDisabled?: boolean;
  router: InjectedRouter;
}

interface IHostQueriesRowProps extends Row {
  original: {
    id?: number;
    should_link_to_hqr?: boolean;
    hostId?: number;
  };
}

const HostQueries = ({
  hostId,
  schedule,
  hostPlatform,
  queryReportsDisabled,
  router,
}: IHostQueriesProps): JSX.Element => {
  const renderEmptyQueriesTab = () => {
    if (hostPlatform === "chrome") {
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

    if (hostPlatform === "ios" || hostPlatform === "ipados") {
      return (
        <EmptyTable
          header="Queries are not supported for this host"
          info={
            <>
              Interested in querying{" "}
              {hostPlatform === "ios" ? "iPhones" : "iPads"}?{" "}
              <CustomLink url={SUPPORT_LINK} text="Let us know" newTab />
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
    () => generateColumnConfigs(hostId, queryReportsDisabled),
    [hostId, queryReportsDisabled]
  );

  const renderHostQueries = () => {
    if (
      !schedule ||
      !schedule.length ||
      hostPlatform === "chrome" ||
      hostPlatform === "ios" ||
      hostPlatform === "ipados"
    ) {
      return renderEmptyQueriesTab();
    }

    return (
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
          disableMultiRowSelect={!queryReportsDisabled} // Removes hover/click state if reports are disabled
          isLoading={false} // loading state handled at parent level
          onSelectSingleRow={onSelectSingleRow}
        />
      </div>
    );
  };

  return (
    <Card
      borderRadiusSize="xxlarge"
      includeShadow
      largePadding
      className={baseClass}
    >
      <p className="card__header">Queries</p>
      {renderHostQueries()}
    </Card>
  );
};

export default HostQueries;
