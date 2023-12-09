import React, { useCallback, useContext, useMemo } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";
import { Row } from "react-table";

import PATHS from "router/paths";
import softwareAPI, {
  ISoftwareQueryKey,
  ISoftwareTitlesResponse,
} from "services/entities/software";
import { AppContext } from "context/app";
import {
  GITHUB_NEW_ISSUE_LINK,
  VULNERABLE_DROPDOWN_OPTIONS,
} from "utilities/constants";
import { getNextLocationPath } from "utilities/helpers";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import TableDataError from "components/DataError";
import TableContainer from "components/TableContainer";
import CustomLink from "components/CustomLink";
import LastUpdatedText from "components/LastUpdatedText";
import { ITableQueryData } from "components/TableContainer/TableContainer";

import EmptySoftwareTable from "../components/EmptySoftwareTable";

import generateSoftwareTitlesTableHeaders from "./SoftwareTitlesTableConfig";

const baseClass = "software-titles";

interface IRowProps extends Row {
  original: {
    id?: number;
  };
}

interface ISoftwareTitlesProps {
  router: InjectedRouter;
  isSoftwareEnabled: boolean;
  query: string;
  perPage: number;
  orderDirection: "asc" | "desc";
  orderKey: string;
  showVulnerableSoftware: boolean;
  currentPage: number;
  teamId?: number;
}

const SoftwareTitles = ({
  router,
  isSoftwareEnabled,
  query,
  perPage,
  orderDirection,
  orderKey,
  showVulnerableSoftware,
  currentPage,
  teamId,
}: ISoftwareTitlesProps) => {
  const { isPremiumTier, isSandboxMode, noSandboxHosts } = useContext(
    AppContext
  );

  // request to get software data
  const {
    data: softwareData,
    isLoading: isSoftwareLoading,
    isError: isSoftwareError,
  } = useQuery<
    ISoftwareTitlesResponse,
    Error,
    ISoftwareTitlesResponse,
    ISoftwareQueryKey[]
  >(
    [
      {
        scope: "software",
        page: currentPage,
        perPage,
        query,
        orderDirection,
        orderKey,
        teamId,
        vulnerable: showVulnerableSoftware,
      },
    ],
    ({ queryKey }) => softwareAPI.getSoftwareTitles(queryKey[0]),
    {
      keepPreviousData: true,
      // stale time can be adjusted if fresher data is desired based on
      // software inventory interval
      staleTime: 30000,
    }
  );

  // determines if a user be able to search in the table
  const searchable =
    isSoftwareEnabled &&
    (!!softwareData?.software_titles || query !== "" || showVulnerableSoftware);

  const softwareTableHeaders = useMemo(
    () =>
      generateSoftwareTitlesTableHeaders(
        router,
        isPremiumTier,
        isSandboxMode,
        teamId
      ),
    [isPremiumTier, isSandboxMode, router, teamId]
  );

  const handleVulnFilterDropdownChange = (isFilterVulnerable: string) => {
    router.replace(
      getNextLocationPath({
        pathPrefix: PATHS.SOFTWARE_TITLES,
        routeTemplate: "",
        queryParams: {
          query,
          teamId,
          orderDirection,
          orderKey,
          vulnerable: isFilterVulnerable,
          page: 0, // resets page index
        },
      })
    );
  };

  const handleRowSelect = (row: IRowProps) => {
    // const hostsBySoftwareParams = {
    //   software_id: row.original.id,
    //   team_id: teamId,
    // };

    // const path = hostsBySoftwareParams
    //   ? `${PATHS.MANAGE_HOSTS}?${buildQueryStringFromParams(
    //       hostsBySoftwareParams
    //     )}`
    //   : PATHS.MANAGE_HOSTS;

    // router.push(path);
    // TODO: navigation to software details page.
    console.log("selectedRow", row.id);
  };

  const generateNewQueryParams = (newTableQuery: ITableQueryData) => {
    return {
      query: newTableQuery.searchQuery,
      teamId,
      orderDirection: newTableQuery.sortDirection,
      orderKey: newTableQuery.sortHeader,
      vulnerable: showVulnerableSoftware.toString(),
      page: newTableQuery.pageIndex,
    };
  };

  // NOTE: this is called once on initial render and every time the query changes
  const onQueryChange = useCallback((newTableQuery: ITableQueryData) => {
    console.log("new query data:", newTableQuery);

    const newRoute = getNextLocationPath({
      pathPrefix: PATHS.SOFTWARE_TITLES,
      routeTemplate: "",
      queryParams: generateNewQueryParams(newTableQuery),
    });

    router.replace(newRoute);

    // if (!isRouteOk || isEqual(newTableQuery, tableQueryData)) {
    //   return;
    // }

    // setTableQueryData({ ...newTableQuery });

    // const {
    //   pageIndex,
    //   searchQuery: newSearchQuery,
    //   sortDirection: newSortDirection,
    // } = newTableQuery;
    // let { sortHeader: newSortHeader } = newTableQuery;

    // pageIndex !== page && setPage(pageIndex);
    // searchQuery !== newSearchQuery && setSearchQuery(newSearchQuery);
    // sortDirection !== newSortDirection &&
    //   setSortDirection(
    //     newSortDirection === "asc" || newSortDirection === "desc"
    //       ? newSortDirection
    //       : DEFAULT_SORT_DIRECTION
    //   );

    // if (isPremiumTier && newSortHeader === "vulnerabilities") {
    //   newSortHeader = "epss_probability";
    // }
    // sortHeader !== newSortHeader && setSortHeader(newSortHeader);

    // // Rebuild queryParams to dispatch new browser location to react-router
    // const newQueryParams: { [key: string]: string | number | undefined } = {};
    // if (!isEmpty(newSearchQuery)) {
    //   newQueryParams.query = newSearchQuery;
    // }
    // newQueryParams.page = pageIndex;
    // newQueryParams.order_key = newSortHeader || DEFAULT_SORT_HEADER;
    // newQueryParams.order_direction =
    //   newSortDirection || DEFAULT_SORT_DIRECTION;

    // newQueryParams.vulnerable = filterVuln ? "true" : undefined;

    // if (teamIdForApi !== undefined) {
    //   newQueryParams.team_id = teamIdForApi;
    // }

    // const locationPath = getNextLocationPath({
    //   pathPrefix: PATHS.SOFTWARE_TITLES,
    //   routeTemplate,
    //   queryParams: newQueryParams,
    // });
    // router.replace(locationPath);
  }, []);

  const renderSoftwareCount = useCallback(() => {
    const lastUpdatedAt = softwareData?.counts_updated_at;

    if (!isSoftwareEnabled || !lastUpdatedAt) {
      return null;
    }

    if (isSoftwareError) {
      return (
        <span className={`${baseClass}__count count-error`}>
          Failed to load software count
        </span>
      );
    }

    const hostCount = softwareData?.count;
    if (hostCount) {
      return (
        <div
          className={`${baseClass}__count ${
            isSoftwareLoading ? "count-loading" : ""
          }`}
        >
          <span>{`${hostCount} software item${
            hostCount === 1 ? "" : "s"
          }`}</span>
          <LastUpdatedText
            lastUpdatedAt={lastUpdatedAt}
            whatToRetrieve={"software"}
          />
        </div>
      );
    }

    return null;
  }, [softwareData, isSoftwareLoading, isSoftwareError, isSoftwareEnabled]);

  const renderVulnFilterDropdown = () => {
    return (
      <Dropdown
        value={showVulnerableSoftware}
        className={`${baseClass}__vuln_dropdown`}
        options={VULNERABLE_DROPDOWN_OPTIONS}
        searchable={false}
        onChange={handleVulnFilterDropdownChange}
        tableFilterDropdown
      />
    );
  };

  const renderTableFooter = () => {
    return (
      <div>
        Seeing unexpected software or vulnerabilities?{" "}
        <CustomLink
          url={GITHUB_NEW_ISSUE_LINK}
          text="File an issue on GitHub"
          newTab
        />
      </div>
    );
  };

  if (isSoftwareError) {
    return <TableDataError />;
  }

  console.log("softwareData", softwareData);

  return (
    <div className={baseClass}>
      <TableContainer
        columns={softwareTableHeaders}
        data={softwareData?.software_titles || []}
        isLoading={isSoftwareLoading}
        resultsTitle={"items"}
        emptyComponent={() => (
          <EmptySoftwareTable
            isSoftwareDisabled={!isSoftwareEnabled}
            isFilterVulnerable={showVulnerableSoftware}
            isSandboxMode={isSandboxMode}
            isCollectingSoftware={false} // TODO: update with new API
            isSearching={query !== ""}
            noSandboxHosts={noSandboxHosts}
          />
        )}
        defaultSortHeader={orderKey}
        defaultSortDirection={orderDirection}
        defaultPageIndex={currentPage}
        defaultSearchQuery={query}
        manualSortBy
        pageSize={perPage}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        disableNextPage // TODO: update with new API
        searchable={searchable}
        inputPlaceHolder="Search by name or vulnerabilities (CVEs)"
        onQueryChange={onQueryChange}
        // additionalQueries serves as a trigger for the useDeepEffect hook
        // to fire onQueryChange for events happeing outside of
        // the TableContainer.
        additionalQueries={showVulnerableSoftware ? "vulnerable" : ""}
        customControl={searchable ? renderVulnFilterDropdown : undefined}
        stackControls
        renderCount={renderSoftwareCount}
        renderFooter={renderTableFooter}
        disableMultiRowSelect
        onSelectSingleRow={handleRowSelect}
      />
    </div>
  );
};

export default SoftwareTitles;
