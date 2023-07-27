/* eslint-disable react/prop-types */
import React, { useContext, useCallback, useMemo } from "react";
import { InjectedRouter } from "react-router";

import { AppContext } from "context/app";
import { IQuery } from "interfaces/query";
import { IEmptyTableProps } from "interfaces/empty_table";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import PATHS from "router/paths";
import { isEmpty } from "lodash";

import { getNextLocationPath } from "utilities/helpers";
import Button from "components/buttons/Button";
import TableContainer from "components/TableContainer";
import CustomLink from "components/CustomLink";
import EmptyTable from "components/EmptyTable";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import generateTableHeaders from "./QueriesTableConfig";

const baseClass = "queries-table";

interface IQueryTableData extends IQuery {
  performance: string;
  platforms: string[];
}
interface IQueriesTableProps {
  queriesList: IQueryTableData[] | null;
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
  isInherited?: boolean;
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
  isLoading,
  onDeleteQueryClick,
  onCreateQueryClick,
  isOnlyObserver,
  isObserverPlus,
  isAnyTeamObserverPlus,
  router,
  queryParams,
  isInherited = false,
}: IQueriesTableProps): JSX.Element | null => {
  const { currentUser } = useContext(AppContext);

  // Functions to avoid race conditions
  const initialSearchQuery = (() => queryParams?.query ?? "")();
  const initialSortHeader = (() =>
    (queryParams?.order_key as "name" | "updated_at" | "author") ?? "name")();
  const initialSortDirection = (() =>
    (queryParams?.order_direction as "asc" | "desc") ?? "asc")();
  const initialPlatform = (() =>
    (queryParams?.platform as "all" | "windows" | "linux" | "darwin") ??
    "all")();
  const initialPage = (() =>
    queryParams && queryParams.page ? parseInt(queryParams?.page, 10) : 0)();

  // Never set as state as URL is source of truth
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

      if (!isEmpty(newSearchQuery)) {
        newQueryParams.query = newSearchQuery;
      }

      newQueryParams.order_key = newSortHeader || DEFAULT_SORT_HEADER;
      newQueryParams.order_direction =
        newSortDirection || DEFAULT_SORT_DIRECTION;
      newQueryParams.platform = platform || DEFAULT_PLATFORM; // must set from URL
      newQueryParams.page = newPageIndex;
      // Reset page number to 0 for new filters
      if (
        newSortDirection !== sortDirection ||
        newSortHeader !== sortHeader ||
        newSearchQuery !== searchQuery
      ) {
        newQueryParams.page = 0;
      }
      newQueryParams.team_id = queryParams?.team_id;
      const locationPath = getNextLocationPath({
        pathPrefix: PATHS.MANAGE_QUERIES,
        queryParams: newQueryParams,
      });

      router?.replace(locationPath);
    },
    [sortHeader, sortDirection, searchQuery, platform, router, page]
  );

  const onClientSidePaginationChange = useCallback(
    (pageIndex: number) => {
      const locationPath = getNextLocationPath({
        pathPrefix: PATHS.MANAGE_QUERIES,
        queryParams: {
          ...queryParams,
          page: pageIndex,
          platform,
          query: searchQuery,
          order_direction: sortDirection,
          order_key: sortHeader,
        },
      });
      router?.replace(locationPath);
    },
    [platform, searchQuery, sortDirection, sortHeader] // Dependencies required for correct variable state
  );

  const emptyState = () => {
    const emptyQueries: IEmptyTableProps = {
      iconName: "empty-queries",
      header: "You don't have any queries",
      info: "A query is a specific question you can ask about your devices.",
    };
    if (searchQuery) {
      delete emptyQueries.iconName;
      emptyQueries.header = "No queries match the current search criteria.";
      emptyQueries.info =
        "Expecting to see queries? Try again in a few seconds as the system catches up.";
    } else if (!isOnlyObserver || isObserverPlus || isAnyTeamObserverPlus) {
      emptyQueries.additionalInfo = (
        <>
          Create a new query, or{" "}
          <CustomLink
            url="https://fleetdm.com/docs/using-fleet/standard-query-library"
            text="import Fleet’s standard query library"
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
          Create new query
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
      />
    );
  };

  const tableHeaders = useMemo(
    () => currentUser && generateTableHeaders({ currentUser, isInherited }),
    [currentUser, isInherited]
  );

  const searchable =
    !(queriesList?.length === 0 && searchQuery === "") && !isInherited;

  return tableHeaders && !isLoading ? (
    <div className={`${baseClass}`}>
      <TableContainer
        disableCount={isInherited}
        resultsTitle="queries"
        columns={tableHeaders}
        data={queriesList}
        filters={{ global: isInherited ? "" : searchQuery }}
        isLoading={isLoading}
        defaultSortHeader={sortHeader || DEFAULT_SORT_HEADER}
        defaultSortDirection={sortDirection || DEFAULT_SORT_DIRECTION}
        defaultSearchQuery={searchQuery}
        defaultPageIndex={page}
        pageSize={DEFAULT_PAGE_SIZE}
        inputPlaceHolder="Search by name"
        onQueryChange={onQueryChange}
        emptyComponent={() =>
          EmptyTable({
            iconName: emptyState().iconName,
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
        customControl={
          searchable && !isInherited ? renderPlatformDropdown : undefined
        }
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
        selectedDropdownFilter={!isInherited ? platform : undefined}
      />
    </div>
  ) : (
    <></>
  );
};

export default QueriesTable;
