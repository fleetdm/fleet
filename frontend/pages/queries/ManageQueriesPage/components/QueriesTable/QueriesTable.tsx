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
  };
}

const DEFAULT_SORT_DIRECTION = "desc";
const DEFAULT_SORT_HEADER = "updated_at";
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
    label: "Linux",
    value: "linux",
    helpText: "Queries that are compatible with Linux operating systems.",
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
];

const QueriesTable = ({
  queriesList,
  isLoading,
  onDeleteQueryClick,
  onCreateQueryClick,
  isOnlyObserver,
  isObserverPlus,
  isAnyTeamObserverPlus,
  queryParams,
  router,
}: IQueriesTableProps): JSX.Element | null => {
  const { currentUser } = useContext(AppContext);

  const initialSearchQuery = (() => {
    let query = "";
    if (queryParams && queryParams.query) {
      query = queryParams.query;
    }
    return query;
  })();

  const initialSortHeader = (() => {
    let sortHeader = "updated_at";
    if (
      queryParams &&
      (queryParams.order_key === "name" ||
        queryParams.order_key === "author_name")
    ) {
      sortHeader = queryParams.order_key;
    }
    return sortHeader;
  })();

  const initialSortDirection = ((): "asc" | "desc" | undefined => {
    let sortDirection = "asc";
    if (queryParams && queryParams.order_direction) {
      sortDirection = queryParams.order_direction;
    }
    return sortDirection as "asc" | "desc" | undefined;
  })();

  const initialPlatform = (() => {
    let platformSelected = "all";
    if (
      queryParams &&
      (queryParams.platform === "windows" ||
        queryParams.platform === "linux" ||
        queryParams.platform === "darwin")
    ) {
      platformSelected = queryParams.platform;
    }
    return platformSelected;
  })();

  const initialPage = (() => {
    let page = 0;
    if (queryParams && queryParams.page) {
      page = parseInt(queryParams?.page, 10) || 0;
    }
    return page;
  })();

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
    () => currentUser && generateTableHeaders({ currentUser }),
    [currentUser]
  );

  const searchable = !(queriesList?.length === 0 && searchQuery === "");

  return tableHeaders && !isLoading ? (
    <div className={`${baseClass}`}>
      <TableContainer
        resultsTitle="queries"
        columns={tableHeaders}
        data={queriesList}
        filters={{ global: searchQuery }}
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
        customControl={searchable ? renderPlatformDropdown : undefined}
        isClientSidePagination
        onClientSidePaginationChange={onClientSidePaginationChange}
        isClientSideFilter
        primarySelectAction={{
          name: "delete query",
          buttonText: "Delete",
          icon: "delete",
          variant: "text-icon",
          onActionButtonClick: onDeleteQueryClick,
        }}
        selectedDropdownFilter={platform}
      />
    </div>
  ) : (
    <></>
  );
};

export default QueriesTable;
