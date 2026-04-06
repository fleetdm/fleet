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
  convertParamsToSnakeCase,
  getPathWithQueryParams,
} from "utilities/url";
import {
  ISoftwareApiParams,
  ISoftwareTitlesResponse,
  ISoftwareVersionsResponse,
} from "services/entities/software";
import { ISoftwareTitle, ISoftwareVersion } from "interfaces/software";

import TableContainer from "components/TableContainer";
import CustomLink from "components/CustomLink";
import LastUpdatedText from "components/LastUpdatedText";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import TableCount from "components/TableContainer/TableCount";
import { SingleValue } from "react-select-5";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";

import EmptySoftwareTable from "pages/SoftwarePage/components/tables/EmptySoftwareTable";

import generateTitlesTableConfig from "./SoftwareLibraryTableConfig";
import {
  ISoftwareDropdownFilterVal,
  SOFTWARE_TITLES_DROPDOWN_OPTIONS,
  buildSoftwareFilterQueryParams,
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

const baseClass = "software-library-table";

const SoftwareTable = ({
  router,
  data,
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
        page: changedParam === "pageIndex" ? newTableQuery.pageIndex : 0,
      };
      // Only include these filters when not on “All teams”
      if (teamId !== undefined) {
        if (softwareFilter === "installableSoftware") {
          newQueryParam.available_for_install = "true";
        }
        if (softwareFilter === "selfServiceSoftware") {
          newQueryParam.self_service = "true";
        }
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

      // Note: There may be no changedParam on initial render, but we still may need
      // to strip unwanted params with generateNewQueryParams so do NOT early return

      const newRoute = getNextLocationPath({
        pathPrefix: PATHS.SOFTWARE_LIBRARY,
        routeTemplate: "",
        queryParams: generateNewQueryParams(newTableQuery, changedParam),
      });

      router.replace(newRoute);
    },
    [determineQueryParamChange, generateNewQueryParams, router]
  );

  let tableData: ISoftwareTitle[] | ISoftwareVersion[] | undefined;
  let generateTableConfig: ITableConfigGenerator;

  if (data === undefined) {
    tableData;
    generateTableConfig = () => [];
  } else if (isSoftwareTitles(data)) {
    tableData = data.software_titles;
    generateTableConfig = generateTitlesTableConfig;
  }

  const softwareTableHeaders = useMemo(() => {
    if (!data) return [];
    return generateTableConfig(router, teamId);
  }, [data, router, teamId]);

  // Determines if a user should be able to filter or search in the table
  const hasData = tableData && tableData.length > 0;
  const hasQuery = query !== "";
  const hasSoftwareFilter = softwareFilter !== "allSoftware";

  const showFilterHeaders =
    isSoftwareEnabled && (hasData || hasQuery || hasSoftwareFilter);

  const handleCustomFilterDropdownChange = (
    value: ISoftwareDropdownFilterVal
  ) => {
    const queryParams: ISoftwareApiParams = {
      query,
      teamId,
      orderDirection,
      orderKey,
      page: 0, // resets page index
      ...buildSoftwareFilterQueryParams(value),
    };

    router.replace(
      getNextLocationPath({
        pathPrefix: PATHS.SOFTWARE_LIBRARY,
        routeTemplate: "",
        queryParams: convertParamsToSnakeCase(queryParams),
      })
    );
  };

  const handleRowSelect = (row: IRowProps) => {
    if (!row.original.id) return;

    const detailsPath = PATHS.SOFTWARE_TITLE_DETAILS(
      row.original.id.toString()
    );

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
      </>
    );
  };

  // TODO: Remake into slider
  const renderCustomControls = () => {
    return (
      <div className={`${baseClass}__filter-controls`}>
        <DropdownWrapper
          name="software-filter"
          value={softwareFilter}
          className={`${baseClass}__software-filter`}
          options={SOFTWARE_TITLES_DROPDOWN_OPTIONS}
          onChange={(newValue: SingleValue<CustomOptionType>) =>
            newValue &&
            handleCustomFilterDropdownChange(
              newValue.value as ISoftwareDropdownFilterVal
            )
          }
          variant="table-filter"
        />
      </div>
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
            isSoftwareDisabled={!isSoftwareEnabled}
            noSearchQuery={query === ""}
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
        // additionalQueries serves as a trigger for the useDeepEffect hook
        // to fire onQueryChange for events happening outside of
        // the TableContainer.
        // This is necessary to remove unwanted query params from the URL
        additionalQueries={softwareFilter}
        customControl={showFilterHeaders ? renderCustomControls : undefined}
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
