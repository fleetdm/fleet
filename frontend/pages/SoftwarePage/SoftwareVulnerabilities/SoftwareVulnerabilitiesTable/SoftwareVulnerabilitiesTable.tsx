/** software/vulnerabilities Vulnerabilities tab > Table */

import React, { useCallback, useContext, useMemo } from "react";
import { InjectedRouter } from "react-router";
import { Row } from "react-table";

import PATHS from "router/paths";

import { AppContext } from "context/app";
import {
  GITHUB_NEW_ISSUE_LINK,
  EXPLOITED_VULNERABILITIES_DROPDOWN_OPTIONS,
} from "utilities/constants";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import CustomLink from "components/CustomLink";
import TableContainer from "components/TableContainer";
import LastUpdatedText from "components/LastUpdatedText";
import { ITableQueryData } from "components/TableContainer/TableContainer";

import EmptySoftwareTable from "pages/SoftwarePage/components/EmptySoftwareTable";
import { IVulnerabilitiesResponse } from "services/entities/vulnerabilities";
import { buildQueryStringFromParams } from "utilities/url";
import { getNextLocationPath } from "utilities/helpers";

import generateTableConfig from "./VulnerabilitiesTableConfig";

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
}: ISoftwareVulnerabilitiesTableProps) => {
  const { isPremiumTier, isSandboxMode, noSandboxHosts } = useContext(
    AppContext
  );

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
          case "exploited":
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
        exploited: showExploitedVulnerabilitiesOnly.toString(),
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
      isSandboxMode,
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
          exploited: isFilterExploited.toString(),
          page: 0, // resets page index
        },
      })
    );
  };

  const handleRowSelect = (row: IRowProps) => {
    const hostsByVulnerabilityParams = {
      cve: row.original.cve,
      team_id: teamId,
    };

    const path = `${PATHS.MANAGE_HOSTS}?${buildQueryStringFromParams(
      hostsByVulnerabilityParams
    )}`;

    router.push(path);
  };

  const getItemsCountText = () => {
    const count = data?.count;
    if (!data?.vulnerabilities || !count) return "";

    return count === 1 ? `${count} vulnerability` : `${count} vulnerabilities`;
  };

  const getLastUpdatedText = () => {
    if (!data?.vulnerabilities || !data?.counts_updated_at) return "";
    return (
      <LastUpdatedText
        lastUpdatedAt={data.counts_updated_at}
        whatToRetrieve={"vulnerabilities"}
      />
    );
  };

  const renderVulnerabilityCount = () => {
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

  const renderExploitedVulnerabilitiesDropdown = () => {
    return (
      <Dropdown
        value={showExploitedVulnerabilitiesOnly}
        className={`${baseClass}__exploited-vulnerabilities-dropdown`}
        options={EXPLOITED_VULNERABILITIES_DROPDOWN_OPTIONS}
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
        data={data?.vulnerabilities ?? []}
        isLoading={isLoading}
        resultsTitle={"items"}
        emptyComponent={() => (
          <EmptySoftwareTable
            isSoftwareDisabled={!isSoftwareEnabled}
            isSandboxMode={isSandboxMode}
            noSandboxHosts={noSandboxHosts}
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
        renderFooter={renderTableFooter}
        disableMultiRowSelect
        onSelectSingleRow={handleRowSelect}
      />
    </div>
  );
};

export default SoftwareVulnerabilitiesTable;
