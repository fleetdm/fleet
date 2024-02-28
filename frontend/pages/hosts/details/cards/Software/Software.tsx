import React, { useCallback, useContext, useMemo } from "react";
import { InjectedRouter } from "react-router";
import { Row } from "react-table";
import PATHS from "router/paths";
import { isEmpty } from "lodash";

import { AppContext } from "context/app";
import { ISoftware } from "interfaces/software";
import { VULNERABLE_DROPDOWN_OPTIONS } from "utilities/constants";
import { buildQueryStringFromParams } from "utilities/url";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import EmptySoftwareTable from "pages/SoftwarePage/components/EmptySoftwareTable";
import { getNextLocationPath } from "utilities/helpers";

import SoftwareVulnCount from "./SoftwareVulnCount";

import {
  generateSoftwareTableHeaders,
  generateSoftwareTableData,
} from "./SoftwareTableConfig";

const baseClass = "host-details";

export interface ITableSoftware extends Omit<ISoftware, "vulnerabilities"> {
  vulnerabilities: string[]; // for client-side search purposes, we only want an array of cve strings
}

interface ISoftwareTableProps {
  isLoading: boolean;
  software: ISoftware[];
  deviceUser?: boolean;
  deviceType?: string;
  isSoftwareEnabled?: boolean;
  router?: InjectedRouter;
  queryParams?: {
    vulnerable?: string;
    page?: string;
    query?: string;
    order_key?: string;
    order_direction?: "asc" | "desc";
  };
  routeTemplate?: string;
  pathname: string;
  pathPrefix: string;
}

interface IRowProps extends Row {
  original: {
    id?: number;
  };
  isSoftwareEnabled?: boolean;
}

const DEFAULT_SORT_DIRECTION = "desc";
const DEFAULT_SORT_HEADER = "name";
const DEFAULT_PAGE_SIZE = 20;

const SoftwareTable = ({
  isLoading,
  software,
  deviceUser,
  deviceType,
  router,
  queryParams,
  routeTemplate,
  pathPrefix,
  pathname,
}: ISoftwareTableProps): JSX.Element => {
  const { isSandboxMode, setFilteredSoftwarePath } = useContext(AppContext);

  // Functions to avoid race conditions
  const initialSearchQuery = (() => queryParams?.query ?? "")();
  const initialSortHeader = (() => queryParams?.order_key ?? "name")();
  const initialSortDirection = (() =>
    (queryParams?.order_direction as "asc" | "desc") ?? "asc")();
  const initialVulnFilter = (() => queryParams?.vulnerable === "true")();
  const initialPage = (() =>
    queryParams && queryParams.page ? parseInt(queryParams?.page, 10) : 0)();

  // Never set as state as URL is source of truth
  const searchQuery = initialSearchQuery;
  const filterVuln = initialVulnFilter;
  const page = initialPage;
  const sortDirection = initialSortDirection;
  const sortHeader = initialSortHeader;

  // TODO: Look into useDebounceCallback with dependencies
  const onQueryChange = useCallback(
    async (newTableQuery: ITableQueryData) => {
      const {
        pageIndex: newPageIndex,
        searchQuery: newSearchQuery,
        sortDirection: newSortDirection,
        sortHeader: newSortHeader,
      } = newTableQuery;

      // Rebuild queryParams to dispatch new browser location to react-router
      const newQueryParams: { [key: string]: string | number | undefined } = {};

      if (!isEmpty(newSearchQuery)) {
        newQueryParams.query = newSearchQuery;
      }

      newQueryParams.order_key = newSortHeader || DEFAULT_SORT_HEADER;
      newQueryParams.order_direction =
        newSortDirection || DEFAULT_SORT_DIRECTION;
      newQueryParams.vulnerable = filterVuln ? "true" : "false"; // must set from URL
      newQueryParams.page = newPageIndex;
      // Reset page number to 0 for new filters
      if (
        newSortDirection !== sortDirection ||
        newSortHeader !== sortHeader ||
        newSearchQuery !== searchQuery
      ) {
        newQueryParams.page = 0;
      }

      const locationPath = getNextLocationPath({
        pathPrefix,
        routeTemplate,
        queryParams: newQueryParams,
      });

      router?.replace(locationPath);
    },
    [sortHeader, sortDirection, searchQuery, filterVuln, router, routeTemplate]
  );

  const onClientSidePaginationChange = useCallback(
    (pageIndex: number) => {
      const locationPath = getNextLocationPath({
        pathPrefix,
        routeTemplate,
        queryParams: {
          ...queryParams,
          page: pageIndex,
          vulnerable: filterVuln ? "true" : "false",
          query: searchQuery,
          order_direction: sortDirection,
          order_key: sortHeader,
        },
      });
      router?.replace(locationPath);
    },
    [filterVuln, searchQuery, sortDirection, sortHeader] // Dependencies required for correct variable state
  );

  const tableSoftware = useMemo(() => generateSoftwareTableData(software), [
    software,
  ]);
  const tableHeaders = useMemo(
    () =>
      generateSoftwareTableHeaders({
        deviceUser,
        router,
        setFilteredSoftwarePath,
        pathname,
      }),
    [deviceUser, router, pathname]
  );

  const handleVulnFilterDropdownChange = (isFilterVulnerable: boolean) => {
    const nextPath = getNextLocationPath({
      pathPrefix,
      routeTemplate,
      queryParams: {
        ...queryParams,
        page: 0,
        vulnerable: isFilterVulnerable.toString(),
      },
    });
    router?.replace(nextPath);
  };

  const handleRowSelect = (row: IRowProps) => {
    if (deviceUser || !router) {
      return;
    }

    const hostsBySoftwareParams = { software_id: row.original.id };

    const path = hostsBySoftwareParams
      ? `${PATHS.MANAGE_HOSTS}?${buildQueryStringFromParams(
          hostsBySoftwareParams
        )}`
      : PATHS.MANAGE_HOSTS;

    router.push(path);
  };

  const renderVulnFilterDropdown = () => {
    return (
      <Dropdown
        value={filterVuln}
        className={`${baseClass}__vuln_dropdown`}
        options={VULNERABLE_DROPDOWN_OPTIONS}
        searchable={false}
        onChange={handleVulnFilterDropdownChange}
        tableFilterDropdown
      />
    );
  };

  return (
    <div className="section section--software">
      <p className="section__header">Software</p>

      {software?.length ? (
        <>
          {software && (
            <SoftwareVulnCount
              softwareList={software}
              deviceUser={deviceUser}
            />
          )}
          {software && (
            <div className={deviceType || ""}>
              <TableContainer
                resultsTitle="software items"
                columnConfigs={tableHeaders}
                data={tableSoftware || []}
                filters={{
                  global: searchQuery,
                  vulnerabilities: filterVuln,
                }}
                isLoading={isLoading}
                defaultSortHeader={sortHeader || DEFAULT_SORT_HEADER}
                defaultSortDirection={sortDirection || DEFAULT_SORT_DIRECTION}
                defaultSearchQuery={searchQuery}
                defaultPageIndex={page}
                pageSize={DEFAULT_PAGE_SIZE}
                inputPlaceHolder="Search by name or vulnerabilities (CVEs)"
                onQueryChange={onQueryChange}
                emptyComponent={() => (
                  <EmptySoftwareTable
                    isFilterVulnerable={filterVuln}
                    isSearching={searchQuery !== ""}
                    isSandboxMode={isSandboxMode}
                  />
                )}
                showMarkAllPages={false}
                isAllPagesSelected={false}
                searchable
                customControl={renderVulnFilterDropdown}
                isClientSidePagination
                onClientSidePaginationChange={onClientSidePaginationChange}
                isClientSideFilter
                disableMultiRowSelect={!deviceUser && !!router} // device user cannot view hosts by software
                onSelectSingleRow={handleRowSelect}
              />
            </div>
          )}
        </>
      ) : (
        <EmptySoftwareTable
          isSandboxMode={isSandboxMode}
          isFilterVulnerable={filterVuln}
        />
      )}
    </div>
  );
};
export default SoftwareTable;
