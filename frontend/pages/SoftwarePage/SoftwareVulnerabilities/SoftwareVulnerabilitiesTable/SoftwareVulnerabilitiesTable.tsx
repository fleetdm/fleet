/** software/vulnerabilities Vulnerabilities tab > Table */

import React, { useCallback, useContext, useMemo } from "react";
import { InjectedRouter } from "react-router";
import { Row } from "react-table";

import PATHS from "router/paths";

import { AppContext } from "context/app";
import { GITHUB_NEW_ISSUE_LINK } from "utilities/constants";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import CustomLink from "components/CustomLink";
import TableContainer from "components/TableContainer";
import LastUpdatedText from "components/LastUpdatedText";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import TableCount from "components/TableContainer/TableCount";

import EmptyVulnerabilitiesTable from "pages/SoftwarePage/components/EmptyVulnerabilitiesTable";
import { isValidCVEFormat } from "pages/SoftwarePage/components/EmptyVulnerabilitiesTable/EmptyVulnerabilitiesTable";

import { IVulnerabilitiesResponse } from "services/entities/vulnerabilities";
import { buildQueryStringFromParams } from "utilities/url";
import { getNextLocationPath } from "utilities/helpers";

import generateTableConfig from "./VulnerabilitiesTableConfig";
import {
  getExploitedVulnerabiltiesDropdownOptions,
  normalizeCVE,
} from "./helpers";

const baseClass = "software-vulnerabilities-table";

interface IRowProps extends Row {
  original: {
    cve?: string;
  };
}

interface ISoftwareVulnerabilitiesTableProps {
  router: InjectedRouter;
  isSoftwareEnabled: boolean;
  data?: IVulnerabilitiesResponse;
  query?: string;
  perPage: number;
  orderDirection: "asc" | "desc";
  orderKey: string;
  showExploitedVulnerabilitiesOnly: boolean;
  currentPage: number;
  teamId?: number;
  isLoading: boolean;
  resetPageIndex: boolean;
}

const SoftwareVulnerabilitiesTable = ({
  router,
  isSoftwareEnabled,
  data,
  query,
  perPage,
  orderDirection,
  orderKey,
  showExploitedVulnerabilitiesOnly,
  currentPage,
  teamId,
  isLoading,
  resetPageIndex,
}: ISoftwareVulnerabilitiesTableProps) => {
  const validQuery = query ? isValidCVEFormat(query) : true;

  // Customer request that turns this table's fuzzy API search
  // into exact match in the UI -- Logic lives here
  // Various empty states live in EmptyVulnerabilitiesTable
  const exactMatchSearchData = (() => {
    // Invalid queries replace any results with no results
    if (!validQuery) {
      return [];
    }

    // No search query renders all results returned from API
    if (!query) {
      return data?.vulnerabilities || [];
    }

    // Query returning results filter the vulnerabilities to return only the exact match
    if (data?.vulnerabilities) {
      const normalizedQuery = normalizeCVE(query);

      return data.vulnerabilities.filter(
        (vulnerability) => normalizeCVE(vulnerability.cve) === normalizedQuery
      );
    }
    return [];
  })();

  const { isPremiumTier } = useContext(AppContext);

  const determineQueryParamChange = useCallback(
    (newTableQuery: ITableQueryData) => {
      const changedEntry = Object.entries(newTableQuery).find(([key, val]) => {
        switch (key) {
          case "sortDirection":
            return val !== orderDirection;
          case "sortHeader":
            return val !== orderKey;
          case "pageIndex":
            return val !== currentPage;
          case "searchQuery":
            return val !== query;
          case "exploit":
            return val !== showExploitedVulnerabilitiesOnly.toString();
          default:
            return false;
        }
      });
      return changedEntry?.[0] ?? "";
    },
    [
      currentPage,
      orderDirection,
      orderKey,
      query,
      showExploitedVulnerabilitiesOnly,
    ]
  );

  const generateNewQueryParams = useCallback(
    (newTableQuery: ITableQueryData, changedParam: string) => {
      return {
        team_id: teamId,
        exploit: showExploitedVulnerabilitiesOnly.toString(),
        query: newTableQuery.searchQuery,
        order_direction: newTableQuery.sortDirection,
        order_key: newTableQuery.sortHeader,
        page: changedParam === "pageIndex" ? newTableQuery.pageIndex : 0,
      };
    },
    [teamId, showExploitedVulnerabilitiesOnly]
  );

  const onQueryChange = useCallback(
    (newTableQuery: ITableQueryData) => {
      // we want to determine which query param has changed in order to
      // reset the page index to 0 if any other param has changed.
      const changedParam = determineQueryParamChange(newTableQuery);

      // if nothing has changed, don't update the route. this can happen when
      // this handler is called on the inital render.
      if (changedParam === "") return;

      const newRoute = getNextLocationPath({
        pathPrefix: PATHS.SOFTWARE_VULNERABILITIES,
        routeTemplate: "",
        queryParams: generateNewQueryParams(newTableQuery, changedParam),
      });

      router.replace(newRoute);
    },
    [determineQueryParamChange, generateNewQueryParams, router]
  );

  // determines if a user be able to search in the table
  const searchable =
    isSoftwareEnabled &&
    (!!data?.vulnerabilities ||
      query !== "" ||
      showExploitedVulnerabilitiesOnly);

  const vulnerabilitiesTableHeaders = useMemo(() => {
    if (!data) return [];
    return generateTableConfig(
      isPremiumTier,
      router,
      {
        includeName: true,
        includeVulnerabilities: true,
        includeIcon: true,
      },
      teamId
    );
  }, [data, router, teamId]);

  const handleExploitedVulnFilterDropdownChange = (
    isFilterExploited: boolean
  ) => {
    router.replace(
      getNextLocationPath({
        pathPrefix: PATHS.SOFTWARE_VULNERABILITIES,
        routeTemplate: "",
        queryParams: {
          query,
          team_id: teamId,
          order_direction: orderDirection,
          order_key: orderKey,
          exploit: isFilterExploited.toString(),
          page: 0, // resets page index
        },
      })
    );
  };

  const handleRowSelect = (row: IRowProps) => {
    const hostsByVulnerabilityParams = {
      vulnerability: row.original.cve,
      team_id: teamId,
    };

    const path = `${PATHS.MANAGE_HOSTS}?${buildQueryStringFromParams(
      hostsByVulnerabilityParams
    )}`;

    router.push(path);
  };

  const renderVulnerabilityCount = () => {
    if (!exactMatchSearchData.length || !data?.count || !validQuery)
      return null;

    // Count without a query is returned from API, but exact match search
    // must show filtered count
    const count = query ? exactMatchSearchData.length : data.count;

    return (
      <>
        <TableCount name="items" count={count} />
        {data?.vulnerabilities && data?.counts_updated_at && (
          <LastUpdatedText
            lastUpdatedAt={data.counts_updated_at}
            customTooltipText={
              <>
                The last time software data was <br />
                updated, including vulnerabilities <br />
                and host counts.
              </>
            }
          />
        )}
      </>
    );
  };

  const renderTableHelpText = () => {
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

  // Exploited vulnerabilities is a premium feature
  const renderExploitedVulnerabilitiesDropdown = () => {
    return (
      <Dropdown
        value={showExploitedVulnerabilitiesOnly}
        className={`${baseClass}__exploited-vulnerabilities-dropdown`}
        options={getExploitedVulnerabiltiesDropdownOptions(isPremiumTier)}
        searchable={false}
        onChange={handleExploitedVulnFilterDropdownChange}
        tableFilterDropdown
      />
    );
  };

  return (
    <div className={baseClass}>
      <TableContainer
        columnConfigs={vulnerabilitiesTableHeaders}
        data={exactMatchSearchData}
        isLoading={isLoading && validQuery}
        resultsTitle={"items"}
        emptyComponent={() => (
          <EmptyVulnerabilitiesTable
            isPremiumTier={isPremiumTier}
            teamId={teamId}
            exploitedFilter={showExploitedVulnerabilitiesOnly}
            isSoftwareDisabled={!isSoftwareEnabled}
            searchQuery={query}
            knownVulnerability={data?.known_vulnerability}
          />
        )}
        defaultSortHeader={orderKey}
        defaultSortDirection={orderDirection}
        defaultPageIndex={currentPage}
        manualSortBy
        pageSize={perPage}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        disableNextPage={!data?.meta.has_next_results}
        searchable={searchable}
        searchQueryColumn="vulnerability"
        inputPlaceHolder="Search by CVE"
        onQueryChange={onQueryChange}
        customControl={
          searchable ? renderExploitedVulnerabilitiesDropdown : undefined
        }
        renderCount={renderVulnerabilityCount}
        renderTableHelpText={renderTableHelpText}
        disableMultiRowSelect
        onSelectSingleRow={handleRowSelect}
        resetPageIndex={resetPageIndex}
      />
    </div>
  );
};

export default SoftwareVulnerabilitiesTable;
