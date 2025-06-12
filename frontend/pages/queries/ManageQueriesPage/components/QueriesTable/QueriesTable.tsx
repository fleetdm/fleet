/* eslint-disable react/prop-types */
import React, { useContext, useCallback, useMemo } from "react";
import { InjectedRouter } from "react-router";
import { Row } from "react-table";
import { SingleValue } from "react-select-5";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { IEmptyTableProps } from "interfaces/empty_table";
import { APP_CONTEXT_ALL_TEAMS_ID } from "interfaces/team";
import { isQueryablePlatform, SelectedPlatform } from "interfaces/platform";
import { IEnhancedQuery } from "interfaces/schedulable_query";
import { getNextLocationPath } from "utilities/helpers";
import { getPathWithQueryParams } from "utilities/url";

import { ITableQueryData } from "components/TableContainer/TableContainer";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
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
  curTeamScopeQueriesPresent: boolean;
  isLoading: boolean;
  onDeleteQueryClick: (selectedTableQueryIds: number[]) => void;
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
  isPremiumTier?: boolean;
}

interface IRowProps extends Row {
  original: {
    id?: number;
  };
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
  curTeamScopeQueriesPresent,
  isLoading,
  onDeleteQueryClick,
  isOnlyObserver,
  isObserverPlus,
  isAnyTeamObserverPlus,
  router,
  queryParams,
  currentTeamId,
  isPremiumTier,
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

  const emptyParams: IEmptyTableProps = {
    graphicName: "empty-queries",
    header: "You don't have any queries",
  };

  if (isPremiumTier) {
    if (
      typeof currentTeamId === "undefined" ||
      currentTeamId === null ||
      currentTeamId === APP_CONTEXT_ALL_TEAMS_ID
    ) {
      emptyParams.header += " that apply to all teams";
    } else {
      emptyParams.header += " that apply to this team";
    }
  }

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
  }

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

  const handleRowSelect = (row: IRowProps) => {
    if (row.original.id) {
      router?.push(
        getPathWithQueryParams(PATHS.QUERY_DETAILS(row.original.id), {
          team_id: currentTeamId,
        })
      );
    }
  };

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
        omitSelectionColumn: !curTeamScopeQueriesPresent,
      }),
    [currentUser, currentTeamId, curTeamScopeQueriesPresent]
  );

  const searchable =
    (totalQueriesCount ?? 0) > 0 || !!targetedPlatformParam || !!searchQuery;

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
          pageIndex={page}
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
          emptyComponent={() => EmptyTable(emptyParams)}
          renderCount={() =>
            ((totalQueriesCount || searchQuery) && (
              <TableCount name="queries" count={totalQueriesCount} />
            )) ||
            null
          }
          inputPlaceHolder="Search by name"
          onQueryChange={onQueryChange}
          searchable={searchable}
          customControl={searchable ? renderPlatformDropdown : undefined}
          disableMultiRowSelect={!curTeamScopeQueriesPresent}
          onClickRow={handleRowSelect}
          selectedDropdownFilter={curTargetedPlatformFilter}
        />
      </div>
    )
  );
};

export default QueriesTable;
