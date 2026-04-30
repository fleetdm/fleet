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
import { getPathWithQueryParams } from "utilities/url";
import {
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

import EmptySoftwareTable from "pages/SoftwarePage/components/tables/EmptySoftwareTable";

import generateInventoryTableConfig from "./SoftwareInventoryTableConfig";
import generateVersionsTableConfig from "./SoftwareVersionsTableConfig";
import {
  ISoftwareVulnFiltersParams,
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
  vulnFilters: ISoftwareVulnFiltersParams;
  currentPage: number;
  teamId?: number;
  isLoading: boolean;
  onAddFiltersClick: () => void;
}

const baseClass = "software-inventory-table";

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
  vulnFilters,
  currentPage,
  teamId,
  isLoading,
  onAddFiltersClick,
}: ISoftwareTableProps) => {
  const currentPath = showVersions
    ? PATHS.SOFTWARE_VERSIONS
    : PATHS.SOFTWARE_INVENTORY;

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
        fleet_id: teamId,
        order_direction: newTableQuery.sortDirection,
        order_key: newTableQuery.sortHeader,
        page:
          changedParam === "pageIndex" || changedParam === "" // Changed param is "" on initial render, so we want to use the page index from the url query for bookmarkability
            ? newTableQuery.pageIndex
            : 0,
        ...buildSoftwareVulnFiltersQueryParams(vulnFilters),
      };

      return newQueryParam;
    },
    [teamId, vulnFilters]
  );

  // NOTE: this is called once on initial render and every time the query changes
  const onQueryChange = useCallback(
    (newTableQuery: ITableQueryData) => {
      // we want to determine which query param has changed in order to
      // reset the page index to 0 if any other param has changed.
      const changedParam = determineQueryParamChange(newTableQuery);

      // Note: There may be no changedParam on initial render, but we still may need
      // to strip unwanted params with generateNewQueryParams so do NOT early return

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
    generateTableConfig = generateInventoryTableConfig;
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
  const vulnFilterDetails = getVulnFilterRenderDetails(vulnFilters);
  const hasVulnFilters = vulnFilterDetails.filterCount > 0;

  const showFilterHeaders =
    isSoftwareEnabled && (hasData || hasQuery || hasVulnFilters);

  const handleShowVersionsToggle = () => {
    const queryParams: Record<string, string | number | boolean | undefined> = {
      query,
      fleet_id: teamId,
      order_direction: orderDirection,
      order_key: orderKey,
      page: 0, // resets page index
      ...buildSoftwareVulnFiltersQueryParams(vulnFilters),
    };

    router.replace(
      getNextLocationPath({
        pathPrefix: showVersions
          ? PATHS.SOFTWARE_INVENTORY
          : PATHS.SOFTWARE_VERSIONS,
        routeTemplate: "",
        queryParams,
      })
    );
  };

  const handleRowSelect = (row: IRowProps) => {
    if (!row.original.id) return;

    const detailsPath = showVersions
      ? PATHS.SOFTWARE_VERSION_DETAILS(row.original.id.toString())
      : PATHS.SOFTWARE_TITLE_DETAILS(row.original.id.toString());

    router.push(getPathWithQueryParams(detailsPath, { fleet_id: teamId }));
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
        <Button variant="inverse" onClick={onAddFiltersClick}>
          <Icon name="filter" />
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
            vulnFilters={vulnFilters}
            isSoftwareDisabled={!isSoftwareEnabled}
            noSearchQuery={query === ""}
            installableSoftwareExists={installableSoftwareExists}
          />
        )}
        defaultSortHeader={orderKey}
        defaultSortDirection={orderDirection}
        pageIndex={currentPage}
        defaultSearchQuery={query}
        manualSortBy
        pageSize={perPage}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        disableNextPage={!data?.meta.has_next_results}
        searchable={showFilterHeaders}
        inputPlaceHolder="Search by name or vulnerability (CVE)"
        onQueryChange={onQueryChange}
        customControl={
          showFilterHeaders ? renderCustomFiltersButton : undefined
        }
        stackControls
        renderCount={renderSoftwareCount}
        renderTableHelpText={renderTableHelpText}
        disableMultiRowSelect
        onSelectSingleRow={handleRowSelect}
      />
    </div>
  );
};

export default SoftwareTable;
