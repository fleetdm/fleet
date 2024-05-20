/**
software/titles Software tab > Table
software/versions Software tab > Table (version toggle on)
*/

import React, { useCallback, useMemo } from "react";
import { InjectedRouter } from "react-router";
import { Row } from "react-table";

import PATHS from "router/paths";
import { getNextLocationPath } from "utilities/helpers";
import { GITHUB_NEW_ISSUE_LINK } from "utilities/constants";
import { buildQueryStringFromParams } from "utilities/url";
import {
  ISoftwareTitlesResponse,
  ISoftwareVersionsResponse,
} from "services/entities/software";
import { ISoftwareTitle, ISoftwareVersion } from "interfaces/software";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import TableContainer from "components/TableContainer";
import Slider from "components/forms/fields/Slider";
import CustomLink from "components/CustomLink";
import LastUpdatedText from "components/LastUpdatedText";
import { ITableQueryData } from "components/TableContainer/TableContainer";

import EmptySoftwareTable from "pages/SoftwarePage/components/EmptySoftwareTable";

import generateTitlesTableConfig from "./SoftwareTitlesTableConfig";
import generateVersionsTableConfig from "./SoftwareVersionsTableConfig";
import {
  ISoftwareDropdownFilterVal,
  SOFTWARE_TITLES_DROPDOWN_OPTIONS,
  SOFTWARE_VERSIONS_DROPDOWN_OPTIONS,
} from "./helpers";

interface IRowProps extends Row {
  original: {
    id?: number;
  };
}

type ITableConfigGenerator = (router: InjectedRouter, teamId?: number) => void;

const isSoftwareTitles = (
  data?: ISoftwareTitlesResponse | ISoftwareVersionsResponse
): data is ISoftwareTitlesResponse => {
  if (!data) return false;
  return (data as ISoftwareTitlesResponse).software_titles !== undefined;
};

interface ISoftwareTableProps {
  router: InjectedRouter;
  data?: ISoftwareTitlesResponse | ISoftwareVersionsResponse;
  showVersions: boolean;
  isSoftwareEnabled: boolean;
  query: string;
  perPage: number;
  orderDirection: "asc" | "desc";
  orderKey: string;
  softwareFilter: ISoftwareDropdownFilterVal;
  currentPage: number;
  teamId?: number;
  isLoading: boolean;
}

const baseClass = "software-table";

const SoftwareTable = ({
  router,
  data,
  showVersions,
  isSoftwareEnabled,
  query,
  perPage,
  orderDirection,
  orderKey,
  softwareFilter,
  currentPage,
  teamId,
  isLoading,
}: ISoftwareTableProps) => {
  const currentPath = showVersions
    ? PATHS.SOFTWARE_VERSIONS
    : PATHS.SOFTWARE_TITLES;

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
          case "pageIndex":
            return val !== currentPage;
          default:
            return false;
        }
      });
      return changedEntry?.[0] ?? "";
    },
    [currentPage, orderDirection, orderKey, query]
  );

  const generateNewQueryParams = useCallback(
    (newTableQuery: ITableQueryData, changedParam: string) => {
      const newQueryParam: Record<string, string | number | undefined> = {
        query: newTableQuery.searchQuery,
        team_id: teamId,
        order_direction: newTableQuery.sortDirection,
        order_key: newTableQuery.sortHeader,
        page: changedParam === "pageIndex" ? newTableQuery.pageIndex : 0,
      };
      if (softwareFilter === "installableSoftware") {
        newQueryParam.available_for_install = true.toString();
      } else {
        newQueryParam.vulnerable = (
          softwareFilter === "vulnerableSoftware"
        ).toString();
      }

      return newQueryParam;
    },
    [softwareFilter, teamId]
  );

  // NOTE: this is called once on initial render and every time the query changes
  const onQueryChange = useCallback(
    (newTableQuery: ITableQueryData) => {
      // we want to determine which query param has changed in order to
      // reset the page index to 0 if any other param has changed.
      const changedParam = determineQueryParamChange(newTableQuery);

      // if nothing has changed, don't update the route. this can happen when
      // this handler is called on the inital render. Can also happen when
      // the filter dropdown is changed. That is handled on the onChange handler
      // for the dropdown.
      if (changedParam === "") return;

      const newRoute = getNextLocationPath({
        pathPrefix: currentPath,
        routeTemplate: "",
        queryParams: generateNewQueryParams(newTableQuery, changedParam),
      });

      router.replace(newRoute);
    },
    [determineQueryParamChange, generateNewQueryParams, router, currentPath]
  );

  let tableData: ISoftwareTitle[] | ISoftwareVersion[] | undefined;
  let generateTableConfig: ITableConfigGenerator;

  if (data === undefined) {
    tableData;
    generateTableConfig = () => [];
  } else if (isSoftwareTitles(data)) {
    tableData = data.software_titles;
    generateTableConfig = generateTitlesTableConfig;
  } else {
    tableData = data.software;
    generateTableConfig = generateVersionsTableConfig;
  }

  const softwareTableHeaders = useMemo(() => {
    if (!data) return [];
    return generateTableConfig(router, teamId);
  }, [generateTableConfig, data, router, teamId]);

  // determines if a user be able to search in the table
  const searchable =
    isSoftwareEnabled &&
    (!!tableData || query !== "" || softwareFilter === "vulnerableSoftware");

  const getItemsCountText = () => {
    const count = data?.count;
    if (!tableData || !count) return "";

    return count === 1 ? `${count} item` : `${count} items`;
  };

  const getLastUpdatedText = () => {
    if (!tableData || !data?.counts_updated_at) return "";
    return (
      <LastUpdatedText
        lastUpdatedAt={data.counts_updated_at}
        whatToRetrieve="software"
      />
    );
  };

  const handleShowVersionsToggle = () => {
    const queryParams: Record<string, string | number | undefined> = {
      query,
      team_id: teamId,
      order_direction: orderDirection,
      order_key: orderKey,
      page: 0, // resets page index
    };

    // if we are currently showing installable titles, we want to switch to
    // all software versions. If not, we want to keep the current filter.
    if (softwareFilter === "installableSoftware") {
      queryParams.vulnerable = "false";
    } else {
      queryParams.vulnerable = (
        softwareFilter === "vulnerableSoftware"
      ).toString();
    }

    router.replace(
      getNextLocationPath({
        pathPrefix: showVersions
          ? PATHS.SOFTWARE_TITLES
          : PATHS.SOFTWARE_VERSIONS,
        routeTemplate: "",
        queryParams,
      })
    );
  };

  const handleVulnFilterDropdownChange = (
    value: ISoftwareDropdownFilterVal
  ) => {
    const queryParams: Record<string, any> = {
      query,
      team_id: teamId,
      order_direction: orderDirection,
      order_key: orderKey,
      page: 0, // resets page index
    };

    if (value === "installableSoftware") {
      queryParams.available_for_install = true;
    } else {
      queryParams.vulnerable = value === "vulnerableSoftware";
    }

    router.replace(
      getNextLocationPath({
        pathPrefix: currentPath,
        routeTemplate: "",
        queryParams,
      })
    );
  };

  const handleRowSelect = (row: IRowProps) => {
    const hostsBySoftwareParams = showVersions
      ? {
          software_version_id: row.original.id,
          team_id: teamId,
        }
      : {
          software_title_id: row.original.id,
          team_id: teamId,
        };

    const path = `${PATHS.MANAGE_HOSTS}?${buildQueryStringFromParams(
      hostsBySoftwareParams
    )}`;

    router.push(path);
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

  const renderCustomFilters = () => {
    const options = showVersions
      ? SOFTWARE_VERSIONS_DROPDOWN_OPTIONS
      : SOFTWARE_TITLES_DROPDOWN_OPTIONS;

    return (
      <div className={`${baseClass}__filter-controls`}>
        <div className={`${baseClass}__version-slider`}>
          {/* div required dropdown form field width bug */}
          <Slider
            value={showVersions}
            onChange={handleShowVersionsToggle}
            inactiveText="Show versions"
            activeText="Show versions"
          />
        </div>
        <Dropdown
          value={softwareFilter}
          className={`${baseClass}__vuln_dropdown`}
          options={options}
          searchable={false}
          onChange={handleVulnFilterDropdownChange}
          tableFilterDropdown
        />
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

  return (
    <div className={baseClass}>
      <TableContainer
        columnConfigs={softwareTableHeaders}
        data={tableData ?? []}
        isLoading={isLoading}
        resultsTitle="items"
        emptyComponent={() => (
          <EmptySoftwareTable
            softwareFilter={softwareFilter}
            isSoftwareDisabled={!isSoftwareEnabled}
            isCollectingSoftware={false} // TODO: update with new API
            isSearching={query !== ""}
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
        disableNextPage={!data?.meta.has_next_results}
        searchable={searchable}
        inputPlaceHolder="Search by name or vulnerabilities (CVEs)"
        onQueryChange={onQueryChange}
        // additionalQueries serves as a trigger for the useDeepEffect hook
        // to fire onQueryChange for events happeing outside of
        // the TableContainer.
        // additionalQueries={softwareFilter}
        customControl={searchable ? renderCustomFilters : undefined}
        stackControls
        renderCount={renderSoftwareCount}
        renderFooter={renderTableFooter}
        disableMultiRowSelect
        onSelectSingleRow={handleRowSelect}
      />
    </div>
  );
};

export default SoftwareTable;
