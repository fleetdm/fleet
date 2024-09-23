/* eslint-disable react/prop-types */
import React, {
  useContext,
  useCallback,
  useMemo,
  useState,
  useEffect,
} from "react";
import { InjectedRouter } from "react-router";

import { AppContext } from "context/app";
import { IEmptyTableProps } from "interfaces/empty_table";
import { SelectedPlatform } from "interfaces/platform";
import { IEnhancedQuery } from "interfaces/schedulable_query";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import { IActionButtonProps } from "components/TableContainer/DataTable/ActionButton/ActionButton";
import PATHS from "router/paths";
import { getNextLocationPath } from "utilities/helpers";
import { checkPlatformCompatibility } from "utilities/sql_tools";
import Button from "components/buttons/Button";
import TableContainer from "components/TableContainer";
import TableCount from "components/TableContainer/TableCount";
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
    platform?: SelectedPlatform;
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

  // Client side filtering bugs fixed with bypassing TableContainer filters
  // queriesState tracks search filter and compatible platform filter
  // to correctly show filtered queries and filtered count
  // isQueryStateLoading prevents flashing of unfiltered count during clientside filtering
  const [queriesState, setQueriesState] = useState<IEnhancedQuery[]>([]);
  const [isQueriesStateLoading, setIsQueriesStateLoading] = useState(true);

  useEffect(() => {
    setIsQueriesStateLoading(true);
    if (queriesList) {
      setQueriesState(
        queriesList.filter((query) => {
          const filterSearchQuery = queryParams?.query
            ? query.name
                .toLowerCase()
                .includes(queryParams?.query.toLowerCase())
            : true;
          const compatiblePlatforms =
            checkPlatformCompatibility(query.query).platforms || [];

          const filterCompatiblePlatform =
            queryParams?.platform && queryParams?.platform !== "all"
              ? compatiblePlatforms.includes(queryParams?.platform)
              : true;

          return filterSearchQuery && filterCompatiblePlatform;
        }) || []
      );
    }
    setIsQueriesStateLoading(false);
  }, [queriesList, queryParams]);

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

  const getEmptyStateParams = useCallback(() => {
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
  }, [
    isAnyTeamObserverPlus,
    isObserverPlus,
    isOnlyObserver,
    onCreateQueryClick,
    searchQuery,
  ]);

  const renderPlatformDropdown = useCallback(() => {
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

    return (
      <Dropdown
        value={platform}
        className={`${baseClass}__platform-dropdown`}
        options={PLATFORM_FILTER_OPTIONS}
        searchable={false}
        onChange={handlePlatformFilterDropdownChange}
        iconName="filter"
      />
    );
  }, [platform, queryParams, router]);

  const renderQueriesCount = useCallback(() => {
    // Fixes flashing incorrect count before clientside filtering
    if (isQueriesStateLoading) {
      return null;
    }

    return <TableCount name="queries" count={queriesState?.length} />;
  }, [queriesState, isQueriesStateLoading]);

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

  const deleteQueryTableActionButtonProps = useMemo(
    () =>
      ({
        name: "delete query",
        buttonText: "Delete",
        iconSvg: "trash",
        variant: "text-icon",
        onActionButtonClick: onDeleteQueryClick,
        // this maintains the existing typing, which is not actually correct
        // TODO - update this object to actually implement IActionButtonProps
      } as IActionButtonProps),
    [onDeleteQueryClick]
  );

  return columnConfigs && !isLoading ? (
    <div className={`${baseClass}`}>
      <TableContainer
        resultsTitle="queries"
        columnConfigs={columnConfigs}
        data={queriesState}
        filters={{ name: trimmedSearchQuery }}
        isLoading={isLoading || isQueriesStateLoading}
        defaultSortHeader={sortHeader || DEFAULT_SORT_HEADER}
        defaultSortDirection={sortDirection || DEFAULT_SORT_DIRECTION}
        defaultSearchQuery={trimmedSearchQuery}
        defaultPageIndex={page}
        pageSize={DEFAULT_PAGE_SIZE}
        inputPlaceHolder="Search by name"
        onQueryChange={onQueryChange}
        emptyComponent={emptyComponent}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        searchable={searchable}
        searchQueryColumn="name"
        customControl={searchable ? renderPlatformDropdown : undefined}
        isClientSidePagination
        onClientSidePaginationChange={onClientSidePaginationChange}
        isClientSideFilter
        primarySelectAction={deleteQueryTableActionButtonProps}
        // TODO - consolidate this functionality within `filters`
        selectedDropdownFilter={platform}
        renderCount={renderQueriesCount}
      />
    </div>
  ) : (
    <></>
  );
};

export default QueriesTable;
