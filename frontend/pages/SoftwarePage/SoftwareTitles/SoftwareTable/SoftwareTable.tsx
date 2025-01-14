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
import {
  buildQueryStringFromParams,
  convertParamsToSnakeCase,
} from "utilities/url";
import {
  ISoftwareApiParams,
  ISoftwareTitlesResponse,
  ISoftwareVersionsResponse,
} from "services/entities/software";
import { ISoftwareTitle, ISoftwareVersion } from "interfaces/software";

import TableContainer from "components/TableContainer";
import Slider from "components/forms/fields/Slider";
import CustomLink from "components/CustomLink";
import LastUpdatedText from "components/LastUpdatedText";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import TableCount from "components/TableContainer/TableCount";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import TooltipWrapper from "components/TooltipWrapper";
import { SingleValue } from "react-select-5";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";

import EmptySoftwareTable from "pages/SoftwarePage/components/EmptySoftwareTable";

import generateTitlesTableConfig from "./SoftwareTitlesTableConfig";
import generateVersionsTableConfig from "./SoftwareVersionsTableConfig";
import {
  ISoftwareDropdownFilterVal,
  ISoftwareVulnFiltersParams,
  SOFTWARE_TITLES_DROPDOWN_OPTIONS,
  buildSoftwareFilterQueryParams,
  buildSoftwareVulnFiltersQueryParams,
  getVulnFilterRenderDetails,
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
  installableSoftwareExists: boolean;
  isSoftwareEnabled: boolean;
  query: string;
  perPage: number;
  orderDirection: "asc" | "desc";
  orderKey: string;
  softwareFilter: ISoftwareDropdownFilterVal;
  vulnFilters: ISoftwareVulnFiltersParams;
  currentPage: number;
  teamId?: number;
  isLoading: boolean;
  resetPageIndex: boolean;
  onAddFiltersClick: () => void;
}

const baseClass = "software-table";

const SoftwareTable = ({
  router,
  data,
  showVersions,
  installableSoftwareExists,
  isSoftwareEnabled,
  query,
  perPage,
  orderDirection,
  orderKey,
  softwareFilter,
  vulnFilters,
  currentPage,
  teamId,
  isLoading,
  resetPageIndex,
  onAddFiltersClick,
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
        ...buildSoftwareVulnFiltersQueryParams(vulnFilters),
      };
      if (softwareFilter === "installableSoftware") {
        newQueryParam.available_for_install = true.toString();
      }
      if (softwareFilter === "selfServiceSoftware") {
        newQueryParam.self_service = true.toString();
      }

      return newQueryParam;
    },
    [softwareFilter, teamId, vulnFilters]
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

  // Determines if a user should be able to filter or search in the table
  const hasData = tableData && tableData.length > 0;
  const hasQuery = query !== "";
  const hasSoftwareFilter = softwareFilter !== "allSoftware";
  const vulnFilterDetails = getVulnFilterRenderDetails(vulnFilters);
  const hasVulnFilters = vulnFilterDetails.filterCount > 0;

  const showFilterHeaders =
    isSoftwareEnabled &&
    (hasData || hasQuery || hasSoftwareFilter || hasVulnFilters);

  const handleShowVersionsToggle = () => {
    const queryParams: Record<string, string | number | boolean | undefined> = {
      query,
      team_id: teamId,
      order_direction: orderDirection,
      order_key: orderKey,
      page: 0, // resets page index
      ...buildSoftwareFilterQueryParams("allSoftware"), // Reset to all software
      ...buildSoftwareVulnFiltersQueryParams(vulnFilters),
    };

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

  const handleCustomFilterDropdownChange = (
    value: ISoftwareDropdownFilterVal
  ) => {
    const queryParams: ISoftwareApiParams = {
      query,
      teamId,
      orderDirection,
      orderKey,
      page: 0, // resets page index
      ...buildSoftwareVulnFiltersQueryParams(vulnFilters),
      ...buildSoftwareFilterQueryParams(value),
    };

    router.replace(
      getNextLocationPath({
        pathPrefix: currentPath,
        routeTemplate: "",
        queryParams: convertParamsToSnakeCase(queryParams),
      })
    );
  };

  const handleRowSelect = (row: IRowProps) => {
    const queryParams = showVersions
      ? buildQueryStringFromParams({
          software_version_id: row.original.id,
          team_id: teamId,
        })
      : buildQueryStringFromParams({
          software_title_id: row.original.id,
          team_id: teamId,
        });

    const path = `${PATHS.MANAGE_HOSTS}?${queryParams}`;

    router.push(path);
  };

  const renderSoftwareCount = () => {
    return (
      <>
        <TableCount name="items" count={data?.count} />
        {tableData && data?.counts_updated_at && (
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
        <Slider
          value={showVersions}
          onChange={handleShowVersionsToggle}
          inactiveText="Show versions"
          activeText="Show versions"
        />
      </>
    );
  };

  const renderCustomControls = () => {
    // Hidden when viewing versions table
    if (showVersions) {
      return null;
    }

    return (
      <div className={`${baseClass}__filter-controls`}>
        <DropdownWrapper
          name="software-filter"
          value={softwareFilter}
          className={`${baseClass}__filter-dropdown`}
          options={SOFTWARE_TITLES_DROPDOWN_OPTIONS}
          onChange={(newValue: SingleValue<CustomOptionType>) =>
            newValue &&
            handleCustomFilterDropdownChange(
              newValue.value as ISoftwareDropdownFilterVal
            )
          }
          iconName="filter"
        />
      </div>
    );
  };

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

  const renderTableHelpText = () => (
    <div>
      Seeing unexpected software or vulnerabilities?{" "}
      <CustomLink
        url={GITHUB_NEW_ISSUE_LINK}
        text="File an issue on GitHub"
        newTab
      />
    </div>
  );

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
            vulnFilters={vulnFilters}
            isSoftwareDisabled={!isSoftwareEnabled}
            noSearchQuery={query === ""}
            installableSoftwareExists={installableSoftwareExists}
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
        searchable={showFilterHeaders}
        inputPlaceHolder="Search by name or vulnerability (CVE)"
        onQueryChange={onQueryChange}
        // additionalQueries serves as a trigger for the useDeepEffect hook
        // to fire onQueryChange for events happening outside of
        // the TableContainer.
        // additionalQueries={softwareFilter}
        customControl={showFilterHeaders ? renderCustomControls : undefined}
        customFiltersButton={
          showFilterHeaders ? renderCustomFiltersButton : undefined
        }
        stackControls
        renderCount={renderSoftwareCount}
        renderTableHelpText={renderTableHelpText}
        disableMultiRowSelect
        onSelectSingleRow={handleRowSelect}
        resetPageIndex={resetPageIndex}
      />
    </div>
  );
};

export default SoftwareTable;
