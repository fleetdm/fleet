import React, { useCallback } from "react";
import { InjectedRouter } from "react-router";

import { IGetHostSoftwareResponse } from "services/entities/hosts";
import { IGetDeviceSoftwareResponse } from "services/entities/device_user";
import { getNextLocationPath } from "utilities/helpers";
import { QueryParams } from "utilities/url";

import { HostPlatform, isAndroid } from "interfaces/platform";

import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";

import EmptySoftwareTable from "pages/SoftwarePage/components/tables/EmptySoftwareTable";
import TableCount from "components/TableContainer/TableCount";
import EmptyTable from "components/EmptyTable";
import CustomLink from "components/CustomLink";
import { SUPPORT_LINK } from "utilities/constants";

const DEFAULT_PAGE_SIZE = 20;

const baseClass = "host-sw-library-table";

interface IHostSoftwareLibraryTableProps {
  tableConfig: any; // TODO: type
  data?: IGetHostSoftwareResponse | IGetDeviceSoftwareResponse;
  platform: HostPlatform;
  isLoading: boolean;
  router: InjectedRouter;
  sortHeader: string;
  sortDirection: "asc" | "desc";
  searchQuery: string;
  page: number;
  pagePath: string;
}

const HostSoftwareLibraryTable = ({
  tableConfig,
  data,
  platform,
  isLoading,
  router,
  sortHeader,
  sortDirection,
  searchQuery,
  page,
  pagePath,
}: IHostSoftwareLibraryTableProps) => {
  const determineQueryParamChange = useCallback(
    (newTableQuery: ITableQueryData) => {
      const changedEntry = Object.entries(newTableQuery).find(([key, val]) => {
        switch (key) {
          case "searchQuery":
            return val !== searchQuery;
          case "sortDirection":
            return val !== sortDirection;
          case "sortHeader":
            return val !== sortHeader;
          case "pageIndex":
            return val !== page;
          default:
            return false;
        }
      });
      return changedEntry?.[0] ?? "";
    },
    [page, searchQuery, sortDirection, sortHeader]
  );

  const generateNewQueryParams = useCallback(
    (newTableQuery: ITableQueryData, changedParam: string) => {
      const newQueryParam: QueryParams = {
        query: newTableQuery.searchQuery,
        order_direction: newTableQuery.sortDirection,
        order_key: newTableQuery.sortHeader,
        page: changedParam === "pageIndex" ? newTableQuery.pageIndex : 0,
      };

      return newQueryParam;
    },
    []
  );

  // TODO: Look into useDebounceCallback with dependencies
  const onQueryChange = useCallback(
    async (newTableQuery: ITableQueryData) => {
      // we want to determine which query param has changed in order to
      // reset the page index to 0 if any other param has changed.
      const changedParam = determineQueryParamChange(newTableQuery);

      // if nothing has changed, don't update the route. this can happen when
      // this handler is called on the inital render. Can also happen when
      // the filter dropdown is changed. That is handled on the onChange handler
      // for the dropdown.
      if (changedParam === "") return;

      const newRoute = getNextLocationPath({
        pathPrefix: pagePath,
        routeTemplate: "",
        queryParams: generateNewQueryParams(newTableQuery, changedParam),
      });

      router.replace(newRoute);
    },
    [determineQueryParamChange, pagePath, generateNewQueryParams, router]
  );

  const count = data?.count || data?.software?.length || 0;
  const isSoftwareNotDetected = count === 0 && searchQuery === "";

  const memoizedSoftwareCount = useCallback(() => {
    if (isSoftwareNotDetected) {
      return null;
    }

    return <TableCount name="items" count={count} />;
  }, [count, isSoftwareNotDetected]);

  const memoizedEmptyComponent = useCallback(() => {
    return <EmptySoftwareTable noSearchQuery={searchQuery === ""} />;
  }, [searchQuery]);

  // Determines if a user should be able to filter or search in the table
  const hasData = data && data.software.length > 0;
  const hasQuery = searchQuery !== "";

  const showFilterHeaders = hasData || hasQuery;

  if (isAndroid(platform)) {
    return (
      <EmptyTable
        header="Installers are not supported for this host"
        info={
          <>
            Interested in installing software on Android hosts?{" "}
            <CustomLink url={SUPPORT_LINK} text="Let us know" newTab />
          </>
        }
      />
    );
  }

  return (
    <div className={baseClass}>
      <TableContainer
        renderCount={memoizedSoftwareCount}
        columnConfigs={tableConfig}
        data={data?.software || []}
        isLoading={isLoading}
        defaultSortHeader={sortHeader}
        defaultSortDirection={sortDirection}
        defaultSearchQuery={searchQuery}
        pageIndex={page}
        disableNextPage={data?.meta.has_next_results === false}
        pageSize={DEFAULT_PAGE_SIZE}
        inputPlaceHolder="Search by name"
        onQueryChange={onQueryChange}
        emptyComponent={memoizedEmptyComponent}
        stackControls
        showMarkAllPages={false}
        isAllPagesSelected={false}
        searchable={showFilterHeaders}
        manualSortBy
        keyboardSelectableRows
      />
    </div>
  );
};

export default HostSoftwareLibraryTable;
