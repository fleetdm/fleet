import React, { useCallback } from "react";
import { Row } from "react-table";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { IOperatingSystemVersion } from "interfaces/operating_system";
import { getNextLocationPath } from "utilities/helpers";
import { getPathWithQueryParams } from "utilities/url";

import { ITableQueryData } from "components/TableContainer/TableContainer";
import TableContainer from "components/TableContainer";

import { generateTableHeaders } from "./OSVersionTableConfig";
import OSVersionsEmptyState from "../OSVersionsEmptyState";
import { parseOSUpdatesCurrentVersionsQueryParams } from "../CurrentVersionSection/CurrentVersionSection";

const baseClass = "os-version-table";

interface IRowProps extends Row {
  original: {
    id?: number;
    name_only?: string;
    version?: string;
  };
}

interface IOSVersionTableProps {
  router: InjectedRouter;
  osVersionData: IOperatingSystemVersion[];
  currentTeamId: number;
  isLoading: boolean;
  queryParams: ReturnType<typeof parseOSUpdatesCurrentVersionsQueryParams>;
  hasNextPage: boolean;
}

const OSVersionTable = ({
  router,
  osVersionData,
  currentTeamId,
  isLoading,
  queryParams,
  hasNextPage,
}: IOSVersionTableProps) => {
  const columns = generateTableHeaders(currentTeamId);

  const determineQueryParamChange = useCallback(
    (newTableQuery: ITableQueryData) => {
      const changedEntry = Object.entries(newTableQuery).find(([key, val]) => {
        switch (key) {
          case "sortDirection":
            return val !== queryParams.order_direction;
          case "sortHeader":
            return val !== queryParams.order_key;
          case "pageIndex":
            return val !== queryParams.page;
          default:
            return false;
        }
      });
      return changedEntry?.[0] ?? "";
    },
    [queryParams.order_direction, queryParams.order_key, queryParams.page]
  );

  const generateNewQueryParams = useCallback(
    (newTableQuery: ITableQueryData, changedParam: string) => {
      const newQueryParam: Record<string, string | number | undefined> = {
        fleet_id: currentTeamId,
        order_direction: newTableQuery.sortDirection,
        order_key: newTableQuery.sortHeader,
        page: changedParam === "pageIndex" ? newTableQuery.pageIndex : 0,
      };

      return newQueryParam;
    },
    [currentTeamId]
  );
  // NOTE: this is called once on initial render and every time the query changes
  const onQueryChange = useCallback(
    (newTableQuery: ITableQueryData) => {
      // we want to determine which query param has changed in order to
      // reset the page index to 0 if any other param has changed.
      const changedParam = determineQueryParamChange(newTableQuery);

      // if nothing has changed, don't update the route. this can happen when
      // this handler is called on the inital render. Can also happen when
      // the filter dropdown is changed. That is handled on the onChange handler
      // for the dropdown.
      if (changedParam === "") return;

      const newRoute = getNextLocationPath({
        pathPrefix: PATHS.CONTROLS_OS_UPDATES,
        routeTemplate: "",
        queryParams: generateNewQueryParams(newTableQuery, changedParam),
      });

      router.replace(newRoute);
    },
    [determineQueryParamChange, generateNewQueryParams, router]
  );

  const onSelectSingleRow = useCallback(
    (row: IRowProps) => {
      const { name_only, version } = row.original;

      const hostsQueryParams = {
        os_name: name_only,
        os_version: version,
        fleet_id: currentTeamId,
      };
      const path = getPathWithQueryParams(PATHS.MANAGE_HOSTS, hostsQueryParams);

      router.push(path);
    },
    [router, currentTeamId]
  );

  return (
    <div className={baseClass}>
      <TableContainer
        columnConfigs={columns}
        data={osVersionData}
        isLoading={isLoading}
        resultsTitle=""
        emptyComponent={OSVersionsEmptyState}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        defaultSortHeader={queryParams.order_key}
        defaultSortDirection={queryParams.order_direction}
        pageIndex={queryParams.page}
        disableTableHeader
        disableCount
        pageSize={queryParams.per_page}
        onQueryChange={onQueryChange}
        disableNextPage={!hasNextPage}
        hideFooter={!hasNextPage && queryParams.page === 0}
        // these 2 properties allow linking on click anywhere in the row
        disableMultiRowSelect
        onSelectSingleRow={onSelectSingleRow}
      />
    </div>
  );
};

export default OSVersionTable;
