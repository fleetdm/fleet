import React, { useCallback, useMemo } from "react";

import { isAndroid, HostPlatform } from "interfaces/platform";
import { IQueryStats } from "interfaces/query_stats";
import { SUPPORT_LINK } from "utilities/constants";
import TableContainer from "components/TableContainer";
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

interface IHostQueriesProps {
  hostId: number;
  schedule?: IQueryStats[];
  hostPlatform: HostPlatform;
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

type EmptyHostQueriesProps = {
  hostPlatform: HostPlatform;
};

const EmptyHostQueries = ({ hostPlatform }: EmptyHostQueriesProps) => {
  const platformActions: Record<string, string> = {
    chrome: "collecting data from your Chromebooks",
    ios: "querying iPhones",
    ipados: "querying iPads",
    android: "querying Android hosts",
  };

  const action = platformActions[hostPlatform];

  if (action) {
    return (
      <div>
        <p className="empty-header">Queries not supported for this host</p>
        <p>
          Interested in {action}?{" "}
          <CustomLink url={SUPPORT_LINK} text="Let us know" newTab />
        </p>
      </div>
    );
  }

  return (
    <div>
      <p className="empty-header">No queries</p>
      <p>Add a query to view custom vitals.</p>
    </div>
  );
};

const HostQueries = ({
  hostId,
  schedule,
  hostPlatform,
  queryReportsDisabled,
  router,
  canAddQuery,
  onClickAddQuery,
}: IHostQueriesProps): JSX.Element => {
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
      hostPlatform === "ipados" ||
      isAndroid(hostPlatform)
    ) {
      return <EmptyHostQueries hostPlatform={hostPlatform} />;
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
