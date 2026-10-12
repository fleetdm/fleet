import React, { useCallback } from "react";
import { InjectedRouter } from "react-router";
import { Row } from "react-table";

import Button from "components/buttons/Button";
import EmptyState from "components/EmptyState";
import Slider from "components/forms/fields/Slider";
import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import TableCount from "components/TableContainer/TableCount";
import {
  HostPlatform,
  PLATFORM_DISPLAY_NAMES,
  isVulnUnsupportedPlatform,
} from "interfaces/platform";
import { IHostSoftware } from "interfaces/software";
import EmptySoftwareTable from "pages/SoftwarePage/components/tables/EmptySoftwareTable";
import { VulnsNotSupported } from "pages/SoftwarePage/components/tables/SoftwareVulnerabilitiesTable/SoftwareVulnerabilitiesTable";
import {
  getFilterRenderDetails,
  ISoftwareFilters,
} from "pages/SoftwarePage/SoftwareInventory/SoftwareInventoryTable/helpers";
import { IGetDeviceSoftwareResponse } from "services/entities/device_user";
import { IGetHostSoftwareResponse } from "services/entities/hosts";

import {
  getHostSoftwareLocationPath,
  isDefaultTypeSelection,
} from "../helpers";

const DEFAULT_PAGE_SIZE = 20;

const baseClass = "host-software-table";

interface IHostSoftwareRowProps extends Row {
  original: IHostSoftware;
}

interface IEmptyComponentProps {
  filters: ISoftwareFilters;
  platform: HostPlatform;
  searchQuery: string;
}

const EmptyComponent = ({
  filters,
  platform,
  searchQuery,
}: IEmptyComponentProps) => {
  const vulnFilterAndNotSupported =
    !!filters.vulnerable && isVulnUnsupportedPlatform(platform);
  return vulnFilterAndNotSupported ? (
    <VulnsNotSupported platformText={PLATFORM_DISPLAY_NAMES[platform]} />
  ) : (
    <EmptySoftwareTable
      filters={filters}
      noSearchQuery={searchQuery === ""}
      platform={platform}
    />
  );
};

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
  filters: ISoftwareFilters;
  teamId?: number;
  /** Current value of the macOS /Applications filter, the inverse of "Show
   * helpers". Only defined on macOS hosts while "macOS app" is selected. */
  macosApplicationsFilter?: boolean;
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
  filters,
  teamId,
  macosApplicationsFilter,
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

  const getLocationPath = useCallback(
    (
      newTableQuery: ITableQueryData,
      changedParam: string,
      macosApplications = macosApplicationsFilter
    ) =>
      getHostSoftwareLocationPath({
        pathname: pagePath,
        platform,
        query: newTableQuery.searchQuery,
        orderKey: newTableQuery.sortHeader,
        orderDirection: newTableQuery.sortDirection,
        page: changedParam === "pageIndex" ? newTableQuery.pageIndex : 0,
        fleetId: teamId,
        macosApplications,
        filters,
      }),
    [filters, pagePath, platform, teamId, macosApplicationsFilter]
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

      router.replace(getLocationPath(newTableQuery, changedParam));
    },
    [determineQueryParamChange, getLocationPath, router]
  );

  const count = data?.count || data?.software?.length || 0;
  const isSoftwareNotDetected = count === 0 && searchQuery === "";

  // Determines if a user should be able to filter or search in the table
  const hasData = data && data.software.length > 0;
  const hasQuery = searchQuery !== "";
  const { isFiltered, buttonText } = getFilterRenderDetails(filters);
  const hasUserFilters =
    isFiltered && !isDefaultTypeSelection(platform, filters);

  // The host reported nothing, or nothing under the platform default.
  const showNoSoftwareFound = isSoftwareNotDetected && !hasUserFilters;
  // Controls are disabled only when there is no selection left to widen.
  const isTrulyEmpty = isSoftwareNotDetected && !isFiltered;

  const showFilterHeaders = isTrulyEmpty || hasData || hasQuery || isFiltered;

  const memoizedSoftwareCount = useCallback(() => {
    return <TableCount name="items" count={count} />;
  }, [count]);

  const onClickMyDeviceRow = useCallback(
    (row: IHostSoftwareRowProps) => {
      onShowInventoryVersions(row.original);
    },
    [onShowInventoryVersions]
  );

  const showHelpersToggle = macosApplicationsFilter !== undefined;

  const onShowHelpersChange = () => {
    router.replace(
      getLocationPath(
        {
          searchQuery,
          sortDirection,
          sortHeader,
          pageIndex: 0,
          pageSize: DEFAULT_PAGE_SIZE,
        },
        "",
        !macosApplicationsFilter
      )
    );
  };

  // TableContainer renders the search after these controls, so it ends up as
  // the right-most control.
  const renderCustomControls = () => (
    <>
      {showHelpersToggle && (
        <Slider
          value={!macosApplicationsFilter}
          onChange={onShowHelpersChange}
          ariaLabel="Show helpers"
          inactiveText="Show helpers"
          activeText="Show helpers"
        />
      )}
      <Button
        variant="secondary"
        onClick={onAddFiltersClick}
        disabled={isTrulyEmpty}
        icon="filter"
      >
        {buttonText}
      </Button>
    </>
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
        pageIndex={page}
        disableNextPage={data?.meta.has_next_results === false}
        pageSize={DEFAULT_PAGE_SIZE}
        inputPlaceHolder="Search by name or vulnerability (CVE)"
        onQueryChange={onQueryChange}
        emptyComponent={() =>
          showNoSoftwareFound ? (
            <EmptyState
              header="No software found"
              info="Expecting to see software? Check back later."
            />
          ) : (
            <EmptyComponent
              filters={filters}
              platform={platform}
              searchQuery={searchQuery}
            />
          )
        }
        customControl={showFilterHeaders ? renderCustomControls : undefined}
        stackControls
        showMarkAllPages={false}
        isAllPagesSelected={false}
        searchable={showFilterHeaders}
        disableSearch={isTrulyEmpty}
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
