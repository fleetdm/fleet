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

  const dataQDOTfleet_maintained_apps: IFleetMaintainedApp[] = [
    {
      id: 2,
      name: "1Password",
      version: "8.10.64",
      platform: "darwin",
      software_title_id: 630,
    },
    {
      id: 1,
      name: "Adobe Acrobat Reader",
      version: "24.005.20414",
      platform: "darwin",
      software_title_id: 2652,
    },
    {
      id: 3,
      name: "Box Drive",
      version: "2.43.205",
      platform: "darwin",
      software_title_id: 2993,
    },
    {
      id: 4,
      name: "Brave",
      version: "1.76.73.0",
      platform: "darwin",
      software_title_id: 2999,
    },
    {
      id: 5,
      name: "Cloudflare WARP",
      version: "2025.1.861.0",
      platform: "darwin",
      software_title_id: 2839,
    },
    {
      id: 6,
      name: "Docker Desktop",
      version: "4.39.0,184744",
      platform: "darwin",
    },
    {
      id: 9,
      name: "Figma",
      version: "125.1.5",
      platform: "darwin",
    },
    {
      id: 8,
      name: "Google Chrome",
      version: "134.0.6998.45",
      platform: "darwin",
      software_title_id: 547,
    },
    {
      id: 10,
      name: "Microsoft Edge",
      version: "134.0.3124.51,bd7a7bb2-a585-4892-8cb1-74c91c53c943",
      platform: "darwin",
      software_title_id: 554,
    },
    {
      id: 11,
      name: "Microsoft Excel",
      version: "16.94.25020927",
      platform: "darwin",
    },
    {
      id: 12,
      name: "Microsoft Teams",
      version: "25031.1205.3471.1031",
      platform: "darwin",
    },
    {
      id: 17,
      name: "Microsoft Visual Studio Code",
      version: "1.98.0",
      platform: "darwin",
    },
    {
      id: 14,
      name: "Microsoft Word",
      version: "16.94.25020927",
      platform: "darwin",
      software_title_id: 2887,
    },
    {
      id: 7,
      name: "Mozilla Firefox",
      version: "136.0",
      platform: "darwin",
    },
    {
      id: 13,
      name: "Notion",
      version: "4.6.1",
      platform: "darwin",
    },
    {
      id: 15,
      name: "Postman",
      version: "11.36.0",
      platform: "darwin",
    },
    {
      id: 16,
      name: "Slack",
      version: "4.42.120",
      platform: "darwin",
    },
    {
      id: 18,
      name: "TeamViewer",
      version: "15.63.4",
      platform: "darwin",
    },
    {
      id: 20,
      name: "WhatsApp",
      version: "2.25.3.81",
      platform: "darwin",
    },
    {
      id: 19,
      name: "Zoom for IT Admins",
      version: "6.3.11.50104",
      platform: "darwin",
      software_title_id: 683,
    },
    {
      id: 22,
      name: "1Password",
      version: "8.10.64",
      platform: "windows",
      software_title_id: 630,
    },
    {
      id: 21,
      name: "Adobe Acrobat Reader",
      version: "24.005.20414",
      platform: "windows",
      software_title_id: 2652,
    },
    {
      id: 23,
      name: "Box Drive",
      version: "2.43.205",
      platform: "windows",
    },
    {
      id: 24,
      name: "Brave",
      version: "1.76.73.0",
      platform: "windows",
      software_title_id: 2999,
    },
    {
      id: 25,
      name: "Cloudflare WARP",
      version: "2025.1.861.0",
      platform: "windows",
    },
    {
      id: 26,
      name: "Docker Desktop",
      version: "4.39.0,184744",
      platform: "windows",
    },
    {
      id: 29,
      name: "Figma",
      version: "125.1.5",
      platform: "windows",
    },
    {
      id: 28,
      name: "Google Chrome",
      version: "134.0.6998.45",
      platform: "windows",
      software_title_id: 547,
    },
    {
      id: 30,
      name: "Microsoft Edge",
      version: "134.0.3124.51,bd7a7bb2-a585-4892-8cb1-74c91c53c943",
      platform: "windows",
    },
    {
      id: 32,
      name: "Microsoft Teams",
      version: "25031.1205.3471.1031",
      platform: "windows",
    },
    {
      id: 37,
      name: "Microsoft Visual Studio Code",
      version: "1.98.0",
      platform: "windows",
    },
    {
      id: 27,
      name: "Mozilla Firefox",
      version: "136.0",
      platform: "windows",
    },
    {
      id: 33,
      name: "Notion",
      version: "4.6.1",
      platform: "windows",
    },
    {
      id: 35,
      name: "Postman",
      version: "11.36.0",
      platform: "windows",
    },
    {
      id: 36,
      name: "Slack",
      version: "4.42.120",
      platform: "windows",
    },
    {
      id: 38,
      name: "TeamViewer",
      version: "15.63.4",
      platform: "windows",
    },
    {
      id: 29,
      name: "Zoom for IT Admins",
      version: "6.3.11.50104",
      platform: "windows",
    },
  ];

  // Note: Serverside filtering will be buggy with pagination if > 20 apps
  // API will need to be refactored to combine macOS/windows apps
  // for correct pagination, sort, and counts when we go over 20 apps
  const combinedAppsByPlatform =
    combineAppsByPlatform(dataQDOTfleet_maintained_apps) ?? [];

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
      defaultPageIndex={currentPage}
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
