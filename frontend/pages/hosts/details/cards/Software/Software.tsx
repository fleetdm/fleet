import React, {
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
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
    page?: string;
    query?: string;
    order_key?: string;
    order_direction?: "asc" | "desc";
  };
  routeTemplate?: string;
  hostId: number;
  pathname: string;
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
  pathname,
}: ISoftwareTableProps): JSX.Element => {
  console.log("Software.tsx queryParams", queryParams);
  const { isSandboxMode, setFilteredSoftwarePath } = useContext(AppContext);

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
    let sortDirection = "asc";

    if (queryParams && queryParams.order_direction) {
      sortDirection = queryParams.order_direction;
    }

    return sortDirection as "asc" | "desc" | undefined;
  })();

  const initialVulnFilter = (() => {
    let isFilteredByVulnerabilities = false;
    console.log("initialVulnFilter queryParams", queryParams);
    if (queryParams && queryParams.vulnerable === "true") {
      isFilteredByVulnerabilities = true;
    }

    return isFilteredByVulnerabilities;
  })();

  const initialPage = (() => {
    let page = 0;

    if (queryParams && queryParams.page) {
      page = parseInt(queryParams?.page, 10) || 0;
    }

    return page;
  })();

  const [searchString, setSearchString] = useState(initialQuery);
  // const [filterVuln, setFilterVuln] = useState(initialVulnFilter);
  const filterVuln = initialVulnFilter;
  const page = initialPage; // Never set page in component as url is source of truth
  const [sortDirection, setSortDirection] = useState<
    "asc" | "desc" | undefined
  >(initialSortDirection);
  const [sortHeader, setSortHeader] = useState(initialSortHeader);
  // const sortDirection = initialSortDirection;
  // const sortHeader = initialSortHeader;
  const [tableQueryData, setTableQueryData] = useState<ITableQueryData>();
  const [resetPageIndex, setResetPageIndex] = useState<boolean>(false);

  useEffect(() => {
    // if (queryParams?.vulnerable !== (filterVuln ? "true" : "false")) {
    //   setFilterVuln(queryParams?.vulnerable === "true");
    // }
    setSearchString(queryParams?.query || "");
  }, [queryParams]);

  // TODO: Look into useDebounceCallback with dependencies
  const onQueryChange = useCallback(
    async (newTableQuery: ITableQueryData) => {
      setTableQueryData({ ...newTableQuery });
      console.log("newTablequery", newTableQuery);
      const {
        pageIndex: newPageIndex,
        searchQuery: newSearchQuery,
        sortDirection: newSortDirection,
        sortHeader: newSortHeader,
      } = newTableQuery;

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
      newQueryParams.page = newPageIndex;
      newQueryParams.order_key = newSortHeader || DEFAULT_SORT_HEADER;
      newQueryParams.order_direction =
        newSortDirection || DEFAULT_SORT_DIRECTION;
      newQueryParams.vulnerable = filterVuln ? "true" : "false"; // must grab from source of truth
      console.log("newQueryParams", newQueryParams);
      const locationPath = getNextLocationPath({
        pathPrefix: PATHS.HOST_SOFTWARE(hostId),
        routeTemplate,
        queryParams: newQueryParams,
      });
      console.log("locationPath", locationPath);
      router?.replace(locationPath);
    },
    [
      tableQueryData,
      sortHeader,
      sortDirection,
      searchString,
      filterVuln,
      router,
      routeTemplate,
    ]
  );

  const onClientSidePaginationChange = useCallback(
    (pageIndex: number) => {
      console.log("onClientSidePaginationChange filterVuln", filterVuln);
      console.log("onClientSidePaginationChange pageIndex", pageIndex);
      console.log("onClientSidePaginationChange queryParams", queryParams);

      const locationPath = getNextLocationPath({
        pathPrefix: PATHS.HOST_SOFTWARE(hostId),
        routeTemplate,
        queryParams: {
          ...queryParams,
          page: pageIndex,
          vulnerable: filterVuln ? "true" : "false",
          query: searchString,
          order_direction: sortDirection,
          order_key: sortHeader,
        },
      });
      router?.replace(locationPath);
    },
    [filterVuln, searchString, sortDirection, sortHeader] // Dependencies required for correct variable state
  );

  // NOTE: used to reset page number to 0 when modifying filters
  // const handleResetPageIndex = () => {
  //   setTableQueryData(
  //     (prevState) =>
  //       ({
  //         ...prevState,
  //         pageIndex: 0,
  //         vulnerable: filterVuln ? "true" : "false",
  //       } as ITableQueryData)
  //   );
  //   setResetPageIndex(true);
  // };

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

  const handleVulnFilterDropdownChange = (isFilterVulnerable: string) => {
    // handleResetPageIndex();
    console.log(
      "handleVulnFilterDropdownChange: isFilterVulnerable",
      isFilterVulnerable
    );
    router?.replace(
      getNextLocationPath({
        pathPrefix: PATHS.HOST_SOFTWARE(hostId),
        routeTemplate,
        queryParams: {
          ...queryParams,
          page: 0,
          vulnerable: isFilterVulnerable,
        },
      })
    );
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
                filters={{
                  global: searchString,
                  vulnerabilities: filterVuln,
                }}
                isLoading={isLoading}
                defaultSortHeader={sortHeader || DEFAULT_SORT_DIRECTION}
                defaultSortDirection={sortDirection || DEFAULT_SORT_DIRECTION}
                defaultPageIndex={page}
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
                resetPageIndex={resetPageIndex}
                searchable
                customControl={renderVulnFilterDropdown}
                isClientSidePagination
                onClientSidePaginationChange={onClientSidePaginationChange}
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
