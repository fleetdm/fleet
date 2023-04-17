import React, {
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
// import { useDebouncedCallback } from "use-debounce";
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
import TableContainer, { ITableQueryData } from "components/TableContainer";
import EmptySoftwareTable from "pages/software/components/EmptySoftwareTable";
import { getNextLocationPath } from "pages/hosts/ManageHostsPage/helpers";

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
    page?: number;
    query?: string;
    order_key?: string;
    order_direction?: "asc" | "desc";
  };
  routeTemplate?: string;
  hostId: number;
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
  hostId,
}: ISoftwareTableProps): JSX.Element => {
  const { isSandboxMode } = useContext(AppContext);

  const initialQuery = (() => {
    let query = "";

    if (queryParams && queryParams.query) {
      query = queryParams.query;
    }

    return query;
  })();

  const initialSortHeader = (() => {
    let sortHeader = "name";

    if (queryParams && queryParams.order_key) {
      sortHeader = queryParams.order_key;
    }

    return sortHeader;
  })();

  const initialSortDirection = ((): "asc" | "desc" | undefined => {
    let sortDirection = "desc";

    if (queryParams && queryParams.order_direction) {
      sortDirection = queryParams.order_direction;
    }

    return sortDirection as "asc" | "desc" | undefined;
  })();

  const initialVulnFilter = (() => {
    let isFilteredByVulnerabilities = false;

    if (queryParams && queryParams.vulnerable === "true") {
      isFilteredByVulnerabilities = true;
    }

    return isFilteredByVulnerabilities;
  })();

  const initialPage = (() => {
    let page = 0;

    if (queryParams && queryParams.page) {
      page = queryParams.page as number;
    }

    return page;
  })();

  const [searchString, setSearchString] = useState(initialQuery);
  const [filterVuln, setFilterVuln] = useState(initialVulnFilter);
  const [page, setPage] = useState(initialPage);
  const [sortDirection, setSortDirection] = useState<
    "asc" | "desc" | undefined
  >(initialSortDirection);
  const [sortHeader, setSortHeader] = useState(initialSortHeader);
  const [tableQueryData, setTableQueryData] = useState<ITableQueryData>();
  const [filters, setFilters] = useState({
    global: searchString,
    vulnerabilities: filterVuln,
    page,
  });

  useEffect(() => {
    setFilters({ global: searchString, vulnerabilities: filterVuln, page });
  }, [searchString, filterVuln, page]);

  const onQueryChange = useCallback(async (newTableQuery: ITableQueryData) => {
    setTableQueryData({ ...newTableQuery });

    const {
      pageIndex,
      searchQuery: newSearchQuery,
      sortDirection: newSortDirection,
      sortHeader: newSortHeader,
    } = newTableQuery;
    console.log("pageIndex", pageIndex);
    console.log("typeof pageIndex", typeof pageIndex);
    console.log("newTableQuery.pageIndex", pageIndex);
    console.log("typeof newTableQuery.pageIndex", typeof pageIndex);
    console.log("newTableQuery", newTableQuery);
    pageIndex !== page && setPage(pageIndex as number);
    searchString !== newSearchQuery && setSearchString(newSearchQuery);
    sortDirection !== newSortDirection &&
      setSortDirection(
        newSortDirection === "asc" || newSortDirection === "desc"
          ? newSortDirection
          : DEFAULT_SORT_DIRECTION
      );

    sortHeader !== newSortHeader && setSortHeader(newSortHeader);

    // Rebuild queryParams to dispatch new browser location to react-router
    const newQueryParams: { [key: string]: string | number | undefined } = {};
    if (!isEmpty(newSearchQuery)) {
      newQueryParams.query = newSearchQuery;
    }
    newQueryParams.page = pageIndex as number;
    newQueryParams.order_key = newSortHeader || DEFAULT_SORT_HEADER;
    newQueryParams.order_direction = newSortDirection || DEFAULT_SORT_DIRECTION;

    newQueryParams.vulnerable = filterVuln ? "true" : undefined;

    console.log("newQueryParams.page", newQueryParams.page);
    const locationPath = getNextLocationPath({
      pathPrefix: PATHS.HOST_SOFTWARE(hostId),
      routeTemplate,
      queryParams: newQueryParams,
    });
    console.log("locationPath", locationPath);
    router?.replace(locationPath);
  }, []);

  const tableSoftware = useMemo(() => generateSoftwareTableData(software), [
    software,
  ]);
  const tableHeaders = useMemo(
    () => generateSoftwareTableHeaders(deviceUser, router),
    [deviceUser, router]
  );

  const onVulnFilterChange = (value: boolean) => {
    setFilterVuln(value);
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
        value={filters.vulnerabilities}
        className={`${baseClass}__vuln_dropdown`}
        options={VULNERABLE_DROPDOWN_OPTIONS}
        searchable={false}
        onChange={onVulnFilterChange}
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
                columns={tableHeaders}
                data={tableSoftware || []}
                // filters={filters}
                isLoading={isLoading}
                defaultSortHeader={sortHeader || DEFAULT_SORT_DIRECTION}
                defaultSortDirection={sortDirection || DEFAULT_SORT_DIRECTION}
                defaultPageIndex={page || 0}
                defaultSearchQuery={searchString}
                inputPlaceHolder={
                  "Search software by name or vulnerabilities ( CVEs)"
                }
                onQueryChange={onQueryChange}
                resultsTitle={"software items"}
                emptyComponent={() => (
                  <EmptySoftwareTable
                    isFilterVulnerable={filterVuln}
                    isSearching={searchString !== ""}
                    isSandboxMode={isSandboxMode}
                  />
                )}
                showMarkAllPages={false}
                isAllPagesSelected={false}
                searchable
                customControl={renderVulnFilterDropdown}
                isClientSidePagination
                pageSize={DEFAULT_PAGE_SIZE}
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
