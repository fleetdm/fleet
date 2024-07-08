import React, { useCallback } from "react";
import { InjectedRouter } from "react-router";

import { IGetHostSoftwareResponse } from "services/entities/hosts";
import { IGetDeviceSoftwareResponse } from "services/entities/device_user";
import { getNextLocationPath } from "utilities/helpers";

import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";

import EmptySoftwareTable from "pages/SoftwarePage/components/EmptySoftwareTable";
import TableCount from "components/TableContainer/TableCount";
import { VULNERABLE_DROPDOWN_OPTIONS } from "utilities/constants";

const DEFAULT_PAGE_SIZE = 20;

const baseClass = "host-software-table";

interface IHostSoftwareTableProps {
  tableConfig: any; // TODO: type
  data?: IGetHostSoftwareResponse | IGetDeviceSoftwareResponse;
  isLoading: boolean;
  router: InjectedRouter;
  sortHeader: string;
  sortDirection: "asc" | "desc";
  searchQuery: string;
  page: number;
  pagePath: string;
  queryParams?: {
    vulnerable?: string;
    page?: string;
    query?: string;
    order_key?: string;
    order_direction?: "asc" | "desc";
  };
  routeTemplate?: string;
  pathPrefix: string;
  vulnerable?: boolean;
}

const HostSoftwareTable = ({
  tableConfig,
  data,
  isLoading,
  router,
  sortHeader,
  sortDirection,
  searchQuery,
  page,
  pagePath,
  routeTemplate,
  pathPrefix,
  vulnerable,
}: IHostSoftwareTableProps) => {
  const handleVulnFilterDropdownChange = (isFilterVulnerable: boolean) => {
    const nextPath = getNextLocationPath({
      pathPrefix,
      routeTemplate,
      queryParams: {
        query: searchQuery,
        order_key: sortHeader,
        order_direction: sortDirection,
        page: 0,
        vulnerable: isFilterVulnerable.toString(),
      },
    });
    console.log("path generated: ", nextPath, isFilterVulnerable);
    router?.replace(nextPath);
  };

  const renderVulnFilterDropdown = () => {
    return (
      <Dropdown
        value={vulnerable}
        className={`${baseClass}__vuln_dropdown`}
        options={VULNERABLE_DROPDOWN_OPTIONS}
        searchable={false}
        onChange={handleVulnFilterDropdownChange}
        tableFilterDropdown
      />
    );
  };
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
      const newQueryParam: Record<
        string,
        string | number | boolean | undefined
      > = {
        query: newTableQuery.searchQuery,
        order_direction: newTableQuery.sortDirection,
        order_key: newTableQuery.sortHeader,
        page: changedParam === "pageIndex" ? newTableQuery.pageIndex : 0,
        vulnerable,
      };

      return newQueryParam;
    },
    [vulnerable]
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

  const count = data?.count || data?.software.length || 0;
  const isSoftwareNotDetected = count === 0 && searchQuery === "";

  const memoizedSoftwareCount = useCallback(() => {
    if (isSoftwareNotDetected) {
      return null;
    }

    return <TableCount name="items" count={count} />;
  }, [count, isSoftwareNotDetected]);

  const memoizedEmptyComponent = useCallback(() => {
    return (
      <EmptySoftwareTable
        isFilterVulnerable={vulnerable}
        isNotDetectingSoftware={searchQuery === ""}
      />
    );
  }, [searchQuery, vulnerable]);

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
        defaultPageIndex={page}
        disableNextPage={data?.meta.has_next_results === false}
        pageSize={DEFAULT_PAGE_SIZE}
        inputPlaceHolder="Search by name"
        onQueryChange={onQueryChange}
        emptyComponent={memoizedEmptyComponent}
        customControl={renderVulnFilterDropdown}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        searchable={!isSoftwareNotDetected}
        manualSortBy
      />
    </div>
  );
};

export default HostSoftwareTable;
