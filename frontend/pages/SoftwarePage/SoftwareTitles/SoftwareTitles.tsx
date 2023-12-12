import React, { useCallback, useContext, useMemo } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";
import { Row } from "react-table";

import PATHS from "router/paths";
import softwareAPI, {
  ISoftwareApiParams,
  ISoftwareTitlesResponse,
} from "services/entities/software";
import { AppContext } from "context/app";
import {
  GITHUB_NEW_ISSUE_LINK,
  VULNERABLE_DROPDOWN_OPTIONS,
} from "utilities/constants";
import { getNextLocationPath } from "utilities/helpers";
import { buildQueryStringFromParams } from "utilities/url";

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

interface ISoftwareTitlesQueryKey extends ISoftwareApiParams {
  scope: "software-titles";
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
    ISoftwareTitlesQueryKey[]
  >(
    [
      {
        scope: "software-titles",
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
    const hostsBySoftwareParams = {
      software_title_id: row.original.id,
      team_id: teamId,
    };

    const path = `${PATHS.MANAGE_HOSTS}?${buildQueryStringFromParams(
      hostsBySoftwareParams
    )}`;

    router.push(path);
  };

  const determineQueryParamChange = useCallback(
    (newTableQuery: ITableQueryData) => {
      const changedEntry = Object.entries(newTableQuery).find(([key, val]) => {
        switch (key) {
          case "searchQuery":
            return val !== query;
          case "sortDirection":
            return val !== orderDirection;
          case "sortHeader":
            return val !== orderKey;
          case "vulnerable":
            return val !== showVulnerableSoftware.toString();
          case "pageIndex":
            return val !== currentPage;
          default:
            return false;
        }
      });
      return changedEntry?.[0] ?? "";
    },
    [currentPage, orderDirection, orderKey, query, showVulnerableSoftware]
  );

  const generateNewQueryParams = useCallback(
    (newTableQuery: ITableQueryData, changedParam: string) => {
      return {
        query: newTableQuery.searchQuery,
        team_id: teamId,
        order_direction: newTableQuery.sortDirection,
        order_key: newTableQuery.sortHeader,
        vulnerable: showVulnerableSoftware.toString(),
        page: changedParam === "pageIndex" ? newTableQuery.pageIndex : 0,
      };
    },
    [showVulnerableSoftware, teamId]
  );

  // NOTE: this is called once on initial render and every time the query changes
  const onQueryChange = useCallback(
    (newTableQuery: ITableQueryData) => {
      // we want to determine which query param has changed in order to
      // reset the page index to 0 if any other param has changed.
      const changedParam = determineQueryParamChange(newTableQuery);

      // if nothing has changed, don't update the route. this can happen when
      // this handler is called on the inital render.
      if (changedParam === "") return;

      const newRoute = getNextLocationPath({
        pathPrefix: PATHS.SOFTWARE_TITLES,
        routeTemplate: "",
        queryParams: generateNewQueryParams(newTableQuery, changedParam),
      });

      router.replace(newRoute);
    },
    [determineQueryParamChange, generateNewQueryParams, router]
  );

  const getItemsCountText = () => {
    const count = softwareData?.count;
    if (!softwareData || !count) return "";

    return count === 1 ? `${count} item` : `${count} items`;
  };

  const getLastUpdatedText = () => {
    if (!softwareData || !softwareData.counts_updated_at) return "";
    return (
      <LastUpdatedText
        lastUpdatedAt={softwareData.counts_updated_at}
        whatToRetrieve={"software"}
      />
    );
  };

  const renderSoftwareCount = () => {
    const itemText = getItemsCountText();
    const lastUpdatedText = getLastUpdatedText();

    if (!itemText) return null;

    return (
      <div className={`${baseClass}__count`}>
        <span>{itemText}</span>
        {lastUpdatedText}
      </div>
    );
  };

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
    return <TableDataError className={`${baseClass}__table-error`} />;
  }

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
        disableNextPage={!softwareData?.meta.has_next_results}
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
