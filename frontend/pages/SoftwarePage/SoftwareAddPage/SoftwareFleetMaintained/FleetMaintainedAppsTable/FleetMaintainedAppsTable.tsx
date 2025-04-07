import React, { useCallback, useMemo } from "react";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { ISoftwareFleetMaintainedAppsResponse } from "services/entities/software";
import { getNextLocationPath } from "utilities/helpers";
import {
  FleetMaintainedAppPlatform,
  ICombinedFMA,
  IFleetMaintainedApp,
} from "interfaces/software";

import TableContainer from "components/TableContainer";
import TableCount from "components/TableContainer/TableCount";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import EmptyTable from "components/EmptyTable";
import CustomLink from "components/CustomLink";

import { generateTableConfig } from "./FleetMaintainedAppsTableConfig";

const baseClass = "fleet-maintained-apps-table";

const EmptyFleetAppsTable = () => (
  <EmptyTable
    graphicName="empty-search-question"
    header="No items match the current search criteria"
    info={
      <>
        Can&apos;t find app?{" "}
        <CustomLink
          newTab
          url="https://fleetdm.com/feature-request"
          text="File an issue on GitHub"
        />
      </>
    }
  />
);

/** Used to convert FleetMaintainedApp API response which has separate entries
 * for Windows FMA and macOS FMA into table friendly format that combines
 * entries for the same app for different platforms */
const combineAppsByPlatform = (
  fmaList: IFleetMaintainedApp[]
): ICombinedFMA[] => {
  const combinedApps: { [name: string]: ICombinedFMA } = {};

  fmaList.forEach((app: IFleetMaintainedApp) => {
    const { name, platform, ...rest } = app;

    if (!combinedApps[name]) {
      combinedApps[name] = { name, macos: null, windows: null };
    }

    if (platform === "darwin") {
      combinedApps[name].macos = {
        platform: platform as FleetMaintainedAppPlatform,
        ...rest,
      };
    } else if (platform === "windows") {
      combinedApps[name].windows = {
        platform: platform as FleetMaintainedAppPlatform,
        ...rest,
      };
    }
  });

  return Object.values(combinedApps);
};

interface IFleetMaintainedAppsTableProps {
  teamId: number;
  isLoading: boolean;
  query: string;
  perPage: number;
  orderDirection: "asc" | "desc";
  orderKey: string;
  currentPage: number;
  router: InjectedRouter;
  data?: ISoftwareFleetMaintainedAppsResponse;
}

interface IRowProps {
  original: IFleetMaintainedApp;
}

const FleetMaintainedAppsTable = ({
  teamId,
  isLoading,
  data,
  router,
  query,
  perPage,
  orderDirection,
  orderKey,
  currentPage,
}: IFleetMaintainedAppsTableProps) => {
  const determineQueryParamChange = useCallback(
    (newTableQuery: ITableQueryData) => {
      const changedEntry = Object.entries(newTableQuery).find(([key, val]) => {
        switch (key) {
          case "searchQuery":
            return val !== query;
          case "sortDirection":
            return val !== orderDirection;
          case "sortHeader":
            return val !== orderKey;
          case "pageIndex":
            return val !== currentPage;
          default:
            return false;
        }
      });
      return changedEntry?.[0] ?? "";
    },
    [currentPage, orderDirection, orderKey, query]
  );

  const generateNewQueryParams = useCallback(
    (newTableQuery: ITableQueryData, changedParam: string) => {
      const newQueryParam: Record<string, string | number | undefined> = {
        query: newTableQuery.searchQuery,
        team_id: teamId,
        order_direction: newTableQuery.sortDirection,
        order_key: newTableQuery.sortHeader,
        page: changedParam === "pageIndex" ? newTableQuery.pageIndex : 0,
      };

      return newQueryParam;
    },
    [teamId]
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
        pathPrefix: PATHS.SOFTWARE_ADD_FLEET_MAINTAINED,
        routeTemplate: "",
        queryParams: generateNewQueryParams(newTableQuery, changedParam),
      });

      router.replace(newRoute);
    },
    [determineQueryParamChange, generateNewQueryParams, router]
  );

  const tableHeadersConfig = useMemo(() => {
    if (!data) return [];
    return generateTableConfig(router, teamId);
  }, [data, router, teamId]);

  // Note: Serverside filtering will be buggy with pagination if > 20 apps
  // API will need to be refactored to combine macOS/windows apps
  // for correct pagination, sort, and counts when we go over 20 apps
  const combinedAppsByPlatform =
    (data && combineAppsByPlatform(data.fleet_maintained_apps)) ?? [];

  const renderCount = () => {
    if (!combinedAppsByPlatform) return null;

    return <TableCount name="items" count={combinedAppsByPlatform.length} />;
  };

  return (
    <TableContainer<IRowProps>
      className={baseClass}
      columnConfigs={tableHeadersConfig}
      data={combinedAppsByPlatform}
      isLoading={isLoading}
      resultsTitle="items"
      emptyComponent={EmptyFleetAppsTable}
      defaultSortHeader={orderKey}
      defaultSortDirection={orderDirection}
      pageIndex={currentPage}
      defaultSearchQuery={query}
      manualSortBy
      pageSize={perPage}
      showMarkAllPages={false}
      isAllPagesSelected={false}
      disableNextPage={!data?.meta.has_next_results}
      searchable
      inputPlaceHolder="Search by name"
      onQueryChange={onQueryChange}
      renderCount={renderCount}
    />
  );
};

export default FleetMaintainedAppsTable;
