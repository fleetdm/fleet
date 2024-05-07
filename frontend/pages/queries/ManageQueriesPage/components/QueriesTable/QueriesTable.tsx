/* eslint-disable react/prop-types */
import React, { useContext, useCallback, useMemo } from "react";
import { InjectedRouter } from "react-router";

import { AppContext } from "context/app";
import { IEmptyTableProps } from "interfaces/empty_table";
import { IEnhancedQuery } from "interfaces/schedulable_query";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import PATHS from "router/paths";
import { getNextLocationPath } from "utilities/helpers";
import Button from "components/buttons/Button";
import TableContainer from "components/TableContainer";
import CustomLink from "components/CustomLink";
import EmptyTable from "components/EmptyTable";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import generateColumnConfigs from "./QueriesTableConfig";

const baseClass = "queries-table";
export interface IQueriesTableProps {
  queriesList: IEnhancedQuery[] | null;
  onlyInheritedQueries: boolean;
  isLoading: boolean;
  onDeleteQueryClick: (selectedTableQueryIds: number[]) => void;
  onCreateQueryClick: () => void;
  isOnlyObserver?: boolean;
  isObserverPlus?: boolean;
  isAnyTeamObserverPlus: boolean;
  router?: InjectedRouter;
  queryParams?: {
    platform?: string;
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
const DEFAULT_PAGE_SIZE = 20;
const DEFAULT_PLATFORM = "all";

const PLATFORM_FILTER_OPTIONS = [
  {
    disabled: false,
    label: "All platforms",
    value: "all",
    helpText: "All queries.",
  },
  {
    disabled: false,
    label: "macOS",
    value: "darwin",
    helpText: "Queries that are compatible with macOS operating systems.",
  },
  {
    disabled: false,
    label: "Windows",
    value: "windows",
    helpText: "Queries that are compatible with Windows operating systems.",
  },
  {
    disabled: false,
    label: "Linux",
    value: "linux",
    helpText: "Queries that are compatible with Linux operating systems.",
  },
  {
    disabled: false,
    label: "ChromeOS",
    value: "chrome",
    helpText: "Queries that are compatible with Chromebooks.",
  },
];

const QueriesTable = ({
  queriesList,
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
  const initialSearchQuery = (() => queryParams?.query ?? "")();
  const initialSortHeader = (() =>
    (queryParams?.order_key as "name" | "updated_at" | "author") ??
    DEFAULT_SORT_HEADER)();
  const initialSortDirection = (() =>
    (queryParams?.order_direction as "asc" | "desc") ??
    DEFAULT_SORT_DIRECTION)();
  const initialPlatform = (() =>
    (queryParams?.platform as "all" | "windows" | "linux" | "darwin") ??
    DEFAULT_PLATFORM)();
  const initialPage = (() =>
    queryParams && queryParams.page ? parseInt(queryParams?.page, 10) : 0)();

  // Source of truth is state held within TableContainer. That state is initialized using URL
  // params, then subsquent updates to that state are pushed to the URL.
  const searchQuery = initialSearchQuery;
  const platform = initialPlatform;
  const page = initialPage;
  const sortDirection = initialSortDirection;
  const sortHeader = initialSortHeader;

  // TODO: Look into useDebounceCallback with dependencies
  const onQueryChange = useCallback(
    async (newTableQuery: ITableQueryData) => {
      const {
        pageIndex: newPageIndex,
        searchQuery: newSearchQuery,
        sortDirection: newSortDirection,
        sortHeader: newSortHeader,
      } = newTableQuery;

      // Rebuild queryParams to dispatch new browser location to react-router
      const newQueryParams: { [key: string]: string | number | undefined } = {};

      // Updates URL params
      newQueryParams.order_key = newSortHeader;
      newQueryParams.order_direction = newSortDirection;
      newQueryParams.platform = platform; // must set from URL
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

      router?.replace(locationPath);
    },
    [sortHeader, sortDirection, searchQuery, platform, router, page]
  );

  const onClientSidePaginationChange = useCallback(
    (pageIndex: number) => {
      const newQueryParams = {
        ...queryParams,
        page: pageIndex, // update main table index
        query: searchQuery,
      };

      const locationPath = getNextLocationPath({
        pathPrefix: PATHS.MANAGE_QUERIES,
        queryParams: newQueryParams,
      });
      router?.replace(locationPath);
    },
    [platform, searchQuery, sortDirection, sortHeader] // Dependencies required for correct variable state
  );

  const emptyState = () => {
    const emptyQueries: IEmptyTableProps = {
      graphicName: "empty-queries",
      header: "You don't have any queries",
    };
    if (searchQuery) {
      delete emptyQueries.graphicName;
      emptyQueries.header = "No matching queries";
      emptyQueries.info = "No queries match the current filters.";
    } else if (!isOnlyObserver || isObserverPlus || isAnyTeamObserverPlus) {
      emptyQueries.additionalInfo = (
        <>
          Create a new query, or{" "}
          <CustomLink
            url="https://fleetdm.com/docs/using-fleet/standard-query-library"
            text="import Fleet's standard query library"
            newTab
          />
        </>
      );
      emptyQueries.primaryButton = (
        <Button
          variant="brand"
          className={`${baseClass}__create-button`}
          onClick={onCreateQueryClick}
        >
          Add query
        </Button>
      );
    }

    return emptyQueries;
  };

  const handlePlatformFilterDropdownChange = (platformSelected: string) => {
    router?.replace(
      getNextLocationPath({
        pathPrefix: PATHS.MANAGE_QUERIES,
        queryParams: {
          ...queryParams,
          page: 0,
          platform: platformSelected,
        },
      })
    );
  };

  const renderPlatformDropdown = () => {
    return (
      <Dropdown
        value={platform}
        className={`${baseClass}__platform-dropdown`}
        options={PLATFORM_FILTER_OPTIONS}
        searchable={false}
        onChange={handlePlatformFilterDropdownChange}
        tableFilterDropdown
      />
    );
  };

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

  const searchable = !(queriesList?.length === 0 && searchQuery === "");

  const trimmedSearchQuery = searchQuery.trim();
  return columnConfigs && !isLoading ? (
    <div className={`${baseClass}`}>
      <TableContainer
        resultsTitle="queries"
        columnConfigs={columnConfigs}
        data={queriesList}
        filters={{ name: trimmedSearchQuery }}
        isLoading={isLoading}
        defaultSortHeader={sortHeader || DEFAULT_SORT_HEADER}
        defaultSortDirection={sortDirection || DEFAULT_SORT_DIRECTION}
        defaultSearchQuery={trimmedSearchQuery}
        defaultPageIndex={page}
        pageSize={DEFAULT_PAGE_SIZE}
        inputPlaceHolder="Search by name"
        onQueryChange={onQueryChange}
        emptyComponent={() =>
          EmptyTable({
            graphicName: emptyState().graphicName,
            header: emptyState().header,
            info: emptyState().info,
            additionalInfo: emptyState().additionalInfo,
            primaryButton: emptyState().primaryButton,
          })
        }
        showMarkAllPages={false}
        isAllPagesSelected={false}
        searchable={searchable}
        searchQueryColumn="name"
        customControl={searchable ? renderPlatformDropdown : undefined}
        isClientSidePagination
        onClientSidePaginationChange={onClientSidePaginationChange}
        isClientSideFilter
        primarySelectAction={{
          name: "delete query",
          buttonText: "Delete",
          iconSvg: "trash",
          variant: "text-icon",
          onActionButtonClick: onDeleteQueryClick,
        }}
        selectedDropdownFilter={platform}
        show0Count
      />
    </div>
  ) : (
    <></>
  );
};

export default QueriesTable;
