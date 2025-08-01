import React, { useCallback } from "react";
import { InjectedRouter } from "react-router";

import { IGetHostSoftwareResponse } from "services/entities/hosts";
import { IGetDeviceSoftwareResponse } from "services/entities/device_user";
import { getNextLocationPath } from "utilities/helpers";
import { QueryParams } from "utilities/url";

import {
  buildSoftwareVulnFiltersQueryParams,
  getVulnFilterRenderDetails,
  ISoftwareVulnFiltersParams,
} from "pages/SoftwarePage/SoftwareTitles/SoftwareTable/helpers";

import {
  ApplePlatform,
  APPLE_PLATFORM_DISPLAY_NAMES,
  HostPlatform,
  isIPadOrIPhone,
  isAndroid,
} from "interfaces/platform";

import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import TooltipWrapper from "components/TooltipWrapper";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

import EmptySoftwareTable from "pages/SoftwarePage/components/tables/EmptySoftwareTable";
import TableCount from "components/TableContainer/TableCount";
import { VulnsNotSupported } from "pages/SoftwarePage/components/tables/SoftwareVulnerabilitiesTable/SoftwareVulnerabilitiesTable";
import { Row } from "react-table";
import { IHostSoftware } from "interfaces/software";
import EmptyTable from "components/EmptyTable";
import CustomLink from "components/CustomLink";
import { SUPPORT_LINK } from "utilities/constants";

const DEFAULT_PAGE_SIZE = 20;

const baseClass = "host-software-table";

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
  vulnFilters: ISoftwareVulnFiltersParams;
  onAddFiltersClick: () => void;
  isMyDevicePage?: boolean;
  onShowInventoryVersions: (software: IHostSoftware) => void;
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
  vulnFilters,
  onAddFiltersClick,
  isMyDevicePage,
  onShowInventoryVersions,
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
      const newQueryParam: QueryParams = {
        query: newTableQuery.searchQuery,
        order_direction: newTableQuery.sortDirection,
        order_key: newTableQuery.sortHeader,
        page: changedParam === "pageIndex" ? newTableQuery.pageIndex : 0,
        ...buildSoftwareVulnFiltersQueryParams(vulnFilters),
      };
      return newQueryParam;
    },
    [vulnFilters]
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
    const vulnFilterAndNotSupported = isIPadOrIPhone(platform);
    return vulnFilterAndNotSupported ? (
      <VulnsNotSupported
        platformText={APPLE_PLATFORM_DISPLAY_NAMES[platform as ApplePlatform]}
      />
    ) : (
      <EmptySoftwareTable noSearchQuery={searchQuery === ""} />
    );
  }, [platform, searchQuery]);

  // Determines if a user should be able to filter or search in the table
  const hasData = data && data.software.length > 0;
  const hasQuery = searchQuery !== "";
  const vulnFilterDetails = getVulnFilterRenderDetails(vulnFilters);
  const hasVulnFilters = vulnFilterDetails.filterCount > 0;

  const showFilterHeaders = hasData || hasQuery || hasVulnFilters;

  const onClickMyDeviceRow = useCallback(
    (row: IHostSoftwareRowProps) => {
      onShowInventoryVersions(row.original);
    },
    [onShowInventoryVersions]
  );

  if (isAndroid(platform)) {
    return (
      <EmptyTable
        header="Software is not supported for this host"
        info={
          <>
            Interested in viewing software for Android hosts?{" "}
            <CustomLink url={SUPPORT_LINK} text="Let us know" newTab />
          </>
        }
      />
    );
  }

  const renderCustomFiltersButton = () => {
    return (
      <TooltipWrapper
        className={`${baseClass}__filters`}
        position="left"
        underline={false}
        showArrow
        tipOffset={12}
        tipContent={vulnFilterDetails.tooltipText}
        disableTooltip={!hasVulnFilters}
      >
        <Button variant="text-link" onClick={onAddFiltersClick}>
          <Icon name="filter" color="core-fleet-blue" />
          <span>{vulnFilterDetails.buttonText}</span>
        </Button>
      </TooltipWrapper>
    );
  };

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
        inputPlaceHolder="Search by name or vulnerability (CVE)"
        onQueryChange={onQueryChange}
        emptyComponent={memoizedEmptyComponent}
        customFiltersButton={
          showFilterHeaders ? renderCustomFiltersButton : undefined
        }
        stackControls
        showMarkAllPages={false}
        isAllPagesSelected={false}
        searchable={showFilterHeaders}
        manualSortBy
        keyboardSelectableRows={isMyDevicePage}
        // my device page row clickability
        disableMultiRowSelect={isMyDevicePage}
        onClickRow={isMyDevicePage ? onClickMyDeviceRow : undefined}
      />
    </div>
  );
};

export default HostSoftwareTable;
