import React, { useCallback } from "react";
import { InjectedRouter } from "react-router";

import { IGetHostSoftwareResponse } from "services/entities/hosts";
import { IGetDeviceSoftwareResponse } from "services/entities/device_user";
import { getNextLocationPath } from "utilities/helpers";
import { QueryParams } from "utilities/url";

import { IHostSoftwareDropdownFilterVal } from "pages/SoftwarePage/SoftwareTitles/SoftwareTable/helpers";

import {
  ApplePlatform,
  APPLE_PLATFORM_DISPLAY_NAMES,
  HostPlatform,
  isIPadOrIPhone,
} from "interfaces/platform";

import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";

import EmptySoftwareTable from "pages/SoftwarePage/components/EmptySoftwareTable";
import TableCount from "components/TableContainer/TableCount";
import { VulnsNotSupported } from "pages/SoftwarePage/components/SoftwareVulnerabilitiesTable/SoftwareVulnerabilitiesTable";
import { Row } from "react-table";
import { IHostSoftware } from "interfaces/software";

const DEFAULT_PAGE_SIZE = 20;

const baseClass = "host-software-table";

export const DROPDOWN_OPTIONS = [
  {
    disabled: false,
    label: "All software",
    value: "allSoftware",
    helpText: "All software installed on your hosts.",
  },
  {
    disabled: false,
    label: "Vulnerable software",
    value: "vulnerableSoftware",
    helpText:
      "All software installed on your hosts with detected vulnerabilities.",
  },
  {
    disabled: false,
    label: "Available for install",
    value: "installableSoftware",
    helpText: "Software that can be installed on your hosts.",
  },
] as const;

interface IHostSoftwareRowProps extends Row {
  original: IHostSoftware;
}
interface IHostSoftwareTableProps {
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
  routeTemplate?: string;
  pathPrefix: string;
  hostSoftwareFilter: IHostSoftwareDropdownFilterVal;
  isMyDevicePage?: boolean;
  onShowSoftwareDetails: (software: IHostSoftware) => void;
}

const HostSoftwareTable = ({
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
  routeTemplate,
  pathPrefix,
  hostSoftwareFilter,
  isMyDevicePage,
  onShowSoftwareDetails,
}: IHostSoftwareTableProps) => {
  const handleFilterDropdownChange = useCallback(
    (val: IHostSoftwareDropdownFilterVal) => {
      const newParams: QueryParams = {
        query: searchQuery,
        order_key: sortHeader,
        order_direction: sortDirection,
        page: 0,
      };

      // mutually exclusive
      if (val === "installableSoftware") {
        newParams.available_for_install = true.toString();
      } else if (val === "vulnerableSoftware") {
        newParams.vulnerable = true.toString();
      }

      const nextPath = getNextLocationPath({
        pathPrefix,
        routeTemplate,
        queryParams: newParams,
      });
      const prevYScroll = window.scrollY;
      setTimeout(() => {
        window.scroll({
          top: prevYScroll,
          behavior: "smooth",
        });
      }, 0);
      router.replace(nextPath);
    },
    [pathPrefix, routeTemplate, router, searchQuery, sortDirection, sortHeader]
  );

  const memoizedFilterDropdown = useCallback(() => {
    return (
      <Dropdown
        value={hostSoftwareFilter}
        options={DROPDOWN_OPTIONS}
        searchable={false}
        onChange={handleFilterDropdownChange}
        iconName="filter"
      />
    );
  }, [handleFilterDropdownChange, hostSoftwareFilter]);

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

      if (hostSoftwareFilter === "vulnerableSoftware") {
        newQueryParam.vulnerable = "true";
      } else if (hostSoftwareFilter === "installableSoftware") {
        newQueryParam.available_for_install = "true";
      }

      return newQueryParam;
    },
    [hostSoftwareFilter]
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
    const vulnFilterAndNotSupported =
      isIPadOrIPhone(platform) && hostSoftwareFilter === "vulnerableSoftware";
    return vulnFilterAndNotSupported ? (
      <VulnsNotSupported
        platformText={APPLE_PLATFORM_DISPLAY_NAMES[platform as ApplePlatform]}
      />
    ) : (
      <EmptySoftwareTable noSearchQuery={searchQuery === ""} />
    );
  }, [hostSoftwareFilter, platform, searchQuery]);

  // Determines if a user should be able to filter or search in the table
  const hasData = data && data.software.length > 0;
  const hasQuery = searchQuery !== "";
  const hasSoftwareFilter = hostSoftwareFilter !== "allSoftware";

  const showFilterHeaders = hasData || hasQuery || hasSoftwareFilter;

  const onClickMyDeviceRow = useCallback(
    (row: IHostSoftwareRowProps) => {
      onShowSoftwareDetails(row.original);
    },
    [onShowSoftwareDetails]
  );

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
        customControl={showFilterHeaders ? memoizedFilterDropdown : undefined}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        searchable={showFilterHeaders}
        manualSortBy
        keyboardSelectableRows
        // my device page row clickability
        disableMultiRowSelect={isMyDevicePage}
        onClickRow={isMyDevicePage ? onClickMyDeviceRow : undefined}
      />
    </div>
  );
};

export default HostSoftwareTable;
