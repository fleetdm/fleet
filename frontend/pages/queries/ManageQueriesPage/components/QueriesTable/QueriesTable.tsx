/* eslint-disable react/prop-types */
import React, { useContext, useCallback, useMemo } from "react";
import { InjectedRouter } from "react-router";

import { AppContext } from "context/app";
import { IEmptyTableProps } from "interfaces/empty_table";
import { isQueryablePlatform, SelectedPlatform } from "interfaces/platform";
import { IEnhancedQuery } from "interfaces/schedulable_query";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import PATHS from "router/paths";
import { getNextLocationPath } from "utilities/helpers";

import { SingleValue } from "react-select-5";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import Button from "components/buttons/Button";
import TableContainer from "components/TableContainer";
import TableCount from "components/TableContainer/TableCount";
import CustomLink from "components/CustomLink";
import EmptyTable from "components/EmptyTable";

import generateColumnConfigs from "./QueriesTableConfig";

const baseClass = "queries-table";
export interface IQueriesTableProps {
  queries: IEnhancedQuery[] | null;
  totalQueriesCount: number | undefined;
  hasNextResults: boolean;
  onlyInheritedQueries: boolean;
  isLoading: boolean;
  onDeleteQueryClick: (selectedTableQueryIds: number[]) => void;
  onCreateQueryClick: () => void;
  isOnlyObserver?: boolean;
  isObserverPlus?: boolean;
  isAnyTeamObserverPlus: boolean;
  router?: InjectedRouter;
  queryParams?: {
    platform?: string; // which targeted platform to filter queries by
    page?: string;
    query?: string;
    order_key?: string;
    order_direction?: "asc" | "desc";
    team_id?: string;
  };
  currentTeamId?: number;
}

const DEFAULT_SORT_DIRECTION = "asc";
const DEFAULT_SORT_HEADER = "name";
// all platforms
const DEFAULT_PLATFORM: SelectedPlatform = "all";

const PLATFORM_FILTER_OPTIONS = [
  {
    disabled: false,
    label: "All platforms",
    value: "all",
  },
  {
    disabled: false,
    label: "macOS",
    value: "darwin",
  },
  {
    disabled: false,
    label: "Windows",
    value: "windows",
  },
  {
    disabled: false,
    label: "Linux",
    value: "linux",
  },
];

const QueriesTable = ({
  queries,
  totalQueriesCount,
  hasNextResults,
  onlyInheritedQueries,
  isLoading,
  onDeleteQueryClick,
  onCreateQueryClick,
  isOnlyObserver,
  isObserverPlus,
  isAnyTeamObserverPlus,
  router,
  queryParams,
  currentTeamId,
}: IQueriesTableProps): JSX.Element | null => {
  const { currentUser } = useContext(AppContext);

  // Functions to avoid race conditions
  // TODO - confirm these are still necessary
  const initialSearchQuery = (() => queryParams?.query ?? "")();
  const initialSortHeader = (() =>
    (queryParams?.order_key as "name" | "updated_at" | "author") ??
    DEFAULT_SORT_HEADER)();
  const initialSortDirection = (() =>
    (queryParams?.order_direction as "asc" | "desc") ??
    DEFAULT_SORT_DIRECTION)();
  const initialPage = (() =>
    queryParams && queryParams.page ? parseInt(queryParams?.page, 10) : 0)();

  // Source of truth is state held within TableContainer. That state is initialized using URL
  // params, then subsequent updates to that state are pushed to the URL.
  // TODO - remove extraneous defintions around these values
  const searchQuery = initialSearchQuery;
  const page = initialPage;
  const sortDirection = initialSortDirection;
  const sortHeader = initialSortHeader;

  const targetedPlatformParam = queryParams?.platform;
  const curTargetedPlatformFilter: SelectedPlatform = isQueryablePlatform(
    targetedPlatformParam
  )
    ? targetedPlatformParam
    : DEFAULT_PLATFORM;

  // TODO: Look into useDebounceCallback with dependencies
  const onQueryChange = useCallback(
    (newTableQuery: ITableQueryData) => {
      const {
        pageIndex: newPageIndex,
        searchQuery: newSearchQuery,
        sortDirection: newSortDirection,
        sortHeader: newSortHeader,
      } = newTableQuery;

      const newQueryParams: Record<string, string | number | undefined> = {};
      newQueryParams.order_key = newSortHeader;
      newQueryParams.order_direction = newSortDirection;
      newQueryParams.platform =
        curTargetedPlatformFilter === "all"
          ? undefined
          : curTargetedPlatformFilter;
      newQueryParams.page = newPageIndex;
      newQueryParams.query = newSearchQuery;
      // Reset page number to 0 for new filters
      if (
        newSortDirection !== sortDirection ||
        newSortHeader !== sortHeader ||
        newSearchQuery !== searchQuery
      ) {
        newQueryParams.page = "0";
      }
      newQueryParams.team_id = queryParams?.team_id;

      const locationPath = getNextLocationPath({
        pathPrefix: PATHS.MANAGE_QUERIES,
        queryParams: { ...queryParams, ...newQueryParams },
      });

      router?.push(locationPath);
    },
    [
      curTargetedPlatformFilter,
      sortDirection,
      sortHeader,
      searchQuery,
      queryParams,
      router,
    ]
  );

  const getEmptyStateParams = useCallback(() => {
    const emptyParams: IEmptyTableProps = {
      graphicName: "empty-queries",
      header: "You don't have any queries",
    };
    if (searchQuery || curTargetedPlatformFilter !== "all") {
      delete emptyParams.graphicName;
      emptyParams.header = "No matching queries";
      emptyParams.info = "No queries match the current filters.";
    } else if (!isOnlyObserver || isObserverPlus || isAnyTeamObserverPlus) {
      emptyParams.additionalInfo = (
        <>
          Create a new query, or{" "}
          <CustomLink
            url="https://fleetdm.com/docs/using-fleet/standard-query-library"
            text="import Fleet's standard query library"
            newTab
          />
        </>
      );
      emptyParams.primaryButton = (
        <Button
          variant="brand"
          className={`${baseClass}__create-button`}
          onClick={onCreateQueryClick}
        >
          Add query
        </Button>
      );
    }

    return emptyParams;
  }, [
    isAnyTeamObserverPlus,
    isObserverPlus,
    isOnlyObserver,
    onCreateQueryClick,
    searchQuery,
  ]);

  const handlePlatformFilterDropdownChange = useCallback(
    (selectedTargetedPlatform: SingleValue<CustomOptionType>) => {
      router?.push(
        getNextLocationPath({
          pathPrefix: PATHS.MANAGE_QUERIES,
          queryParams: {
            ...queryParams,
            page: 0,
            platform:
              // separate URL & API 0-values of `platform` (undefined) from dropdown
              // 0-value of "all"
              selectedTargetedPlatform?.value === "all"
                ? undefined
                : selectedTargetedPlatform?.value,
          },
        })
      );
    },
    [queryParams, router]
  );

  const renderPlatformDropdown = useCallback(() => {
    return (
      <DropdownWrapper
        name="platform-dropdown"
        value={curTargetedPlatformFilter}
        className={`${baseClass}__platform-dropdown`}
        options={PLATFORM_FILTER_OPTIONS}
        onChange={handlePlatformFilterDropdownChange}
        variant="table-filter"
      />
    );
  }, [curTargetedPlatformFilter, queryParams, router]);

  const columnConfigs = useMemo(
    () =>
      currentUser &&
      generateColumnConfigs({
        currentUser,
        currentTeamId,
        omitSelectionColumn: onlyInheritedQueries,
      }),
    [currentUser, currentTeamId, onlyInheritedQueries]
  );

  const searchable =
    (totalQueriesCount ?? 0) > 0 ||
    !!curTargetedPlatformFilter ||
    !!searchQuery;

  const emptyComponent = useCallback(() => {
    const {
      graphicName,
      header,
      info,
      additionalInfo,
      primaryButton,
    } = getEmptyStateParams();
    return EmptyTable({
      graphicName,
      header,
      info,
      additionalInfo,
      primaryButton,
    });
  }, [getEmptyStateParams]);

  const trimmedSearchQuery = searchQuery.trim();

  return (
    columnConfigs && (
      <div className={`${baseClass}`}>
        <TableContainer
          resultsTitle="queries"
          columnConfigs={columnConfigs}
          data={queries}
          // won't ever actually be loading, see render condition above
          isLoading={isLoading}
          defaultSortHeader={sortHeader || DEFAULT_SORT_HEADER}
          defaultSortDirection={sortDirection || DEFAULT_SORT_DIRECTION}
          defaultSearchQuery={trimmedSearchQuery}
          defaultPageIndex={page}
          disableNextPage={!hasNextResults}
          showMarkAllPages={false}
          isAllPagesSelected={false}
          primarySelectAction={{
            name: "delete query",
            buttonText: "Delete",
            iconSvg: "trash",
            variant: "text-icon",
            onActionButtonClick: onDeleteQueryClick,
          }}
          emptyComponent={emptyComponent}
          renderCount={() => (
            <TableCount name="queries" count={totalQueriesCount} />
          )}
          inputPlaceHolder="Search by name"
          onQueryChange={onQueryChange}
          searchable={searchable}
          customControl={searchable ? renderPlatformDropdown : undefined}
          selectedDropdownFilter={curTargetedPlatformFilter}
        />
      </div>
    )
  );
};

export default QueriesTable;
