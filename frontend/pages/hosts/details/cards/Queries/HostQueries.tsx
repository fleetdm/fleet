import React, { useCallback, useMemo } from "react";

import { isAndroid } from "interfaces/platform";
import { IQueryStats } from "interfaces/query_stats";
import { SUPPORT_LINK } from "utilities/constants";
import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";
import Card from "components/Card";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import CardHeader from "components/CardHeader";
import Icon from "components/Icon";
import PATHS from "router/paths";
import { InjectedRouter } from "react-router";
import { Row } from "react-table";

import {
  generateColumnConfigs,
  generateDataSet,
} from "./HostQueriesTableConfig";

const baseClass = "host-queries-card";
const PAGE_SIZE = 4;
const QUERIES_NOT_SUPPORTED = "Queries are not supported for this host";

interface IHostQueriesProps {
  hostId: number;
  schedule?: IQueryStats[];
  hostPlatform: string;
  queryReportsDisabled?: boolean;
  router: InjectedRouter;
  canAddQuery?: boolean;
  onClickAddQuery: () => void;
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
  canAddQuery,
  onClickAddQuery,
}: IHostQueriesProps): JSX.Element => {
  const renderEmptyQueriesTab = () => {
    if (hostPlatform === "chrome") {
      return (
        <EmptyTable
          header={QUERIES_NOT_SUPPORTED}
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
          header={QUERIES_NOT_SUPPORTED}
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

    if (isAndroid(hostPlatform)) {
      return (
        <EmptyTable
          header={QUERIES_NOT_SUPPORTED}
          info={
            <>
              Interested in querying Android hosts?{" "}
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
        disablePagination={tableData.length <= PAGE_SIZE}
        pageSize={PAGE_SIZE}
        isClientSidePagination
        disableCount
        disableMultiRowSelect={!queryReportsDisabled} // Removes hover/click state if reports are disabled
        isLoading={false} // loading state handled at parent level
        onSelectSingleRow={onSelectSingleRow}
      />
    );
  };

  return (
    <Card className={baseClass} borderRadiusSize="xxlarge" paddingSize="xlarge">
      <div className={`${baseClass}__header`}>
        <CardHeader header="Queries" />
        {canAddQuery && (
          <Button variant="inverse" onClick={onClickAddQuery} size="small">
            <Icon name="plus" />
            Add query
          </Button>
        )}
      </div>

      {renderHostQueries()}
    </Card>
  );
};

export default HostQueries;
