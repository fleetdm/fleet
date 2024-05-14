import React, { useCallback } from "react";
import { InjectedRouter } from "react-router";

import { IGetHostSoftwareResponse } from "services/entities/hosts";
import { IGetDeviceSoftwareResponse } from "services/entities/device_user";
import { getNextLocationPath } from "utilities/helpers";

import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";

import EmptySoftwareTable from "pages/SoftwarePage/components/EmptySoftwareTable";

const DEFAULT_PAGE_SIZE = 20;

const baseClass = "host-software-table";

interface IHostSoftwareTableProps {
  tableConfig: any; // TODO: type
  data: IGetHostSoftwareResponse | IGetDeviceSoftwareResponse;
  isLoading: boolean;
  router: InjectedRouter;
  sortHeader: string;
  sortDirection: "asc" | "desc";
  searchQuery: string;
  page: number;
  pagePath: string;
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
}: IHostSoftwareTableProps) => {
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
      const newQueryParam: Record<string, string | number | undefined> = {
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

  const getItemsCountText = () => {
    const count = data?.count;
    if (!data?.software?.length || !count) return "";

    return count === 1 ? `${count} software item` : `${count} software items`;
  };

  const renderSoftwareCount = () => {
    const itemText = getItemsCountText();

    if (!itemText) return null;

    return (
      <div className={`${baseClass}__count`}>
        <span>{itemText}</span>
      </div>
    );
  };

  return (
    <div className={baseClass}>
      <TableContainer
        renderCount={renderSoftwareCount}
        resultsTitle="software items"
        columnConfigs={tableConfig}
        data={data.software}
        isLoading={isLoading}
        defaultSortHeader={sortHeader}
        defaultSortDirection={sortDirection}
        defaultSearchQuery={searchQuery}
        defaultPageIndex={page}
        pageSize={DEFAULT_PAGE_SIZE}
        inputPlaceHolder="Search by name"
        onQueryChange={onQueryChange}
        emptyComponent={() => (
          <EmptySoftwareTable isSearching={searchQuery !== ""} />
        )}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        searchable
        manualSortBy
      />
    </div>
  );
};

export default HostSoftwareTable;
