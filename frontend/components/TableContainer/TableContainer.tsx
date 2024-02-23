import React, { useState, useCallback, useRef, useEffect } from "react";
import classnames from "classnames";
import { Row } from "react-table";
import ReactTooltip from "react-tooltip";
import useDeepEffect from "hooks/useDeepEffect";

import SearchField from "components/forms/fields/SearchField";
// @ts-ignore
import Pagination from "components/Pagination";
import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";
import { COLORS } from "styles/var/colors";

import DataTable from "./DataTable/DataTable";
import TableContainerUtils from "./TableContainerUtils";
import { IActionButtonProps } from "./DataTable/ActionButton/ActionButton";

export interface ITableQueryData {
  pageIndex: number;
  pageSize: number;
  searchQuery: string;
  sortHeader: string;
  sortDirection: string;
  /**  Only used for showing inherited policies table */
  showInheritedTable?: boolean;
  /** Only used for sort/query changes to inherited policies table */
  editingInheritedTable?: boolean;
}
interface IRowProps extends Row {
  original: {
    id?: number;
    os_version_id?: string; // Required for onSelectSingleRow of SoftwareOSTable.tsx
    cve?: string; // Required for onSelectSingleRow of SoftwareVulnerabilityTable.tsx
  };
}

interface ITableContainerProps {
  columnConfigs: any; // TODO: Figure out type
  data: any; // TODO: Figure out type
  isLoading: boolean;
  manualSortBy?: boolean;
  defaultSortHeader?: string;
  defaultSortDirection?: string;
  defaultSearchQuery?: string;
  defaultPageIndex?: number;
  /** Button visible above the table container next to search bar */
  actionButton?: IActionButtonProps;
  inputPlaceHolder?: string;
  disableActionButton?: boolean;
  disableMultiRowSelect?: boolean;
  resultsTitle: string;
  resultsHtml?: JSX.Element;
  additionalQueries?: string;
  emptyComponent: React.ElementType;
  className?: string;
  showMarkAllPages: boolean;
  isAllPagesSelected: boolean; // TODO: make dependent on showMarkAllPages
  toggleAllPagesSelected?: any; // TODO: an event type and make it dependent on showMarkAllPages
  searchable?: boolean;
  wideSearch?: boolean;
  disablePagination?: boolean;
  disableNextPage?: boolean; // disableNextPage is a temporary workaround for the case
  // where the number of items on the last page is equal to the page size.
  // The old page controls for server-side pagination render a no results screen
  // with a back button. This fix instead disables the next button in that case.
  disableCount?: boolean;
  /** Main button after selecting a row */
  primarySelectAction?: IActionButtonProps;
  /** Secondary button/s after selecting a row */
  secondarySelectActions?: IActionButtonProps[]; // TODO: Combine with primarySelectAction as these are all rendered in the same spot
  filteredCount?: number;
  searchToolTipText?: string;
  searchQueryColumn?: string;
  selectedDropdownFilter?: string;
  isClientSidePagination?: boolean;
  /** Used to set URL to correct path and include page query param */
  onClientSidePaginationChange?: (pageIndex: number) => void;
  isClientSideFilter?: boolean;
  /** isMultiColumnFilter is used to preserve the table headers
  in lieu of displaying the empty component when client-side filtering yields zero results */
  isMultiColumnFilter?: boolean;
  disableHighlightOnHover?: boolean;
  pageSize?: number;
  onQueryChange?:
    | ((queryData: ITableQueryData) => void)
    | ((queryData: ITableQueryData) => number);
  customControl?: () => JSX.Element;
  stackControls?: boolean;
  onSelectSingleRow?: (value: Row | IRowProps) => void;
  /** Use for clientside filtering: Use key global for filtering on any column, or use column id as key */
  filters?: Record<string, string | number | boolean>;
  renderCount?: () => JSX.Element | null;
  renderFooter?: () => JSX.Element | null;
  setExportRows?: (rows: Row[]) => void;
  resetPageIndex?: boolean;
  disableTableHeader?: boolean;
}

const baseClass = "table-container";

const DEFAULT_PAGE_SIZE = 20;
const DEFAULT_PAGE_INDEX = 0;

const TableContainer = ({
  columnConfigs,
  data,
  filters,
  isLoading,
  manualSortBy = false,
  defaultSearchQuery = "",
  defaultPageIndex = DEFAULT_PAGE_INDEX,
  defaultSortHeader = "name",
  defaultSortDirection = "asc",
  inputPlaceHolder = "Search",
  additionalQueries,
  resultsTitle,
  resultsHtml,
  emptyComponent,
  className,
  disableActionButton,
  disableMultiRowSelect = false,
  actionButton,
  showMarkAllPages,
  isAllPagesSelected,
  toggleAllPagesSelected,
  searchable,
  wideSearch,
  disablePagination,
  disableNextPage,
  disableCount,
  primarySelectAction,
  secondarySelectActions,
  filteredCount,
  searchToolTipText,
  isClientSidePagination,
  onClientSidePaginationChange,
  isClientSideFilter,
  isMultiColumnFilter,
  disableHighlightOnHover,
  pageSize = DEFAULT_PAGE_SIZE,
  selectedDropdownFilter,
  searchQueryColumn,
  onQueryChange,
  customControl,
  stackControls,
  onSelectSingleRow,
  renderCount,
  renderFooter,
  setExportRows,
  resetPageIndex,
  disableTableHeader,
}: ITableContainerProps): JSX.Element => {
  const [searchQuery, setSearchQuery] = useState(defaultSearchQuery);
  const [sortHeader, setSortHeader] = useState(defaultSortHeader || "");
  const [sortDirection, setSortDirection] = useState(
    defaultSortDirection || ""
  );
  const [pageIndex, setPageIndex] = useState<number>(defaultPageIndex);
  const [clientFilterCount, setClientFilterCount] = useState<number>();

  // Client side pagination is being overridden to previous page without this
  useEffect(() => {
    if (isClientSidePagination && pageIndex !== defaultPageIndex) {
      setPageIndex(defaultPageIndex);
    }
  }, [defaultPageIndex, pageIndex, isClientSidePagination]);

  const prevPageIndex = useRef(0);

  const wrapperClasses = classnames(baseClass, className);

  const EmptyComponent = emptyComponent;

  const onSortChange = useCallback(
    (id?: string, isDesc?: boolean) => {
      if (id === undefined) {
        setSortHeader(defaultSortHeader || "");
        setSortDirection(defaultSortDirection || "");
      } else {
        setSortHeader(id);
        const direction = isDesc ? "desc" : "asc";
        setSortDirection(direction);
      }
    },
    [defaultSortHeader, defaultSortDirection, setSortHeader, setSortDirection]
  );

  const onSearchQueryChange = (value: string) => {
    setSearchQuery(value);
  };

  const hasPageIndexChangedRef = useRef(false);
  const onPaginationChange = useCallback(
    (newPage: number) => {
      if (!isClientSidePagination) {
        setPageIndex(newPage);
        hasPageIndexChangedRef.current = true;
      }
    },
    [hasPageIndexChangedRef, isClientSidePagination]
  );

  // NOTE: used to reset page number to 0 when modifying filters
  useEffect(() => {
    if (pageIndex !== 0 && resetPageIndex && !isClientSidePagination) {
      onPaginationChange(0);
    }
  }, [resetPageIndex, pageIndex, isClientSidePagination]);

  const onResultsCountChange = useCallback((resultsCount: number) => {
    setClientFilterCount(resultsCount);
  }, []);

  useDeepEffect(() => {
    if (!onQueryChange) {
      return;
    }

    const queryData = {
      searchQuery,
      sortHeader,
      sortDirection,
      pageSize,
      pageIndex,
    };

    if (prevPageIndex.current === pageIndex) {
      setPageIndex(0);
    }

    // NOTE: used to reset page number to 0 when modifying filters
    const newPageIndex = onQueryChange(queryData);
    if (newPageIndex === 0) {
      setPageIndex(0);
    }

    prevPageIndex.current = pageIndex;
  }, [
    searchQuery,
    sortHeader,
    sortDirection,
    pageSize,
    pageIndex,
    additionalQueries,
  ]);

  // TODO: refactor existing components relying on displayCount to use renderCount pattern
  const displayCount = useCallback((): any => {
    if (typeof filteredCount === "number") {
      return filteredCount;
    } else if (typeof clientFilterCount === "number") {
      return clientFilterCount;
    }
    return data?.length || 0;
  }, [filteredCount, clientFilterCount, data]);

  const renderPagination = useCallback(() => {
    if (disablePagination || isClientSidePagination) {
      return null;
    }
    return (
      <Pagination
        resultsOnCurrentPage={data.length}
        currentPage={pageIndex}
        resultsPerPage={pageSize}
        onPaginationChange={onPaginationChange}
        disableNextPage={disableNextPage}
      />
    );
  }, [
    data,
    disablePagination,
    isClientSidePagination,
    disableNextPage,
    pageIndex,
    pageSize,
    onPaginationChange,
  ]);

  const opacity = isLoading ? { opacity: 0.4 } : { opacity: 1 };

  return (
    <div className={wrapperClasses}>
      {wideSearch && searchable && (
        <div className={`${baseClass}__search-input wide-search`}>
          <SearchField
            placeholder={inputPlaceHolder}
            defaultValue={searchQuery}
            onChange={onSearchQueryChange}
          />
        </div>
      )}
      {!disableTableHeader && (
        <div
          className={`${baseClass}__header ${
            stackControls ? "stack-table-controls" : ""
          }`}
        >
          <div
            className={`${baseClass}__header-left ${
              stackControls ? "stack-table-controls" : ""
            }`}
          >
            <span className="results-count">
              {renderCount && (
                <div
                  className={`${baseClass}__results-count ${
                    stackControls ? "stack-table-controls" : ""
                  }`}
                  style={opacity}
                >
                  {renderCount()}
                </div>
              )}
              {!renderCount &&
              !disableCount &&
              (isMultiColumnFilter || displayCount()) ? (
                <div
                  className={`${baseClass}__results-count ${
                    stackControls ? "stack-table-controls" : ""
                  }`}
                  style={opacity}
                >
                  {TableContainerUtils.generateResultsCountText(
                    resultsTitle,
                    displayCount()
                  )}
                  {resultsHtml}
                </div>
              ) : (
                <div />
              )}
            </span>
            <span className="controls">
              {actionButton && !actionButton.hideButton && (
                <Button
                  disabled={disableActionButton}
                  onClick={actionButton.onActionButtonClick}
                  variant={actionButton.variant || "brand"}
                  className={`${baseClass}__table-action-button`}
                >
                  <>
                    {actionButton.buttonText}
                    {actionButton.iconSvg && (
                      <Icon name={actionButton.iconSvg} />
                    )}
                  </>
                </Button>
              )}
              {customControl && customControl()}
            </span>
          </div>

          {/* Render search bar only if not empty component */}
          {searchable && !wideSearch && (
            <div className={`${baseClass}__search`}>
              <div
                className={`${baseClass}__search-input ${
                  stackControls ? "stack-table-controls" : ""
                }`}
                data-tip
                data-for="search-tooltip"
                data-tip-disable={!searchToolTipText}
              >
                <SearchField
                  placeholder={inputPlaceHolder}
                  defaultValue={searchQuery}
                  onChange={onSearchQueryChange}
                />
              </div>
              <ReactTooltip
                effect="solid"
                backgroundColor={COLORS["tooltip-bg"]}
                id="search-tooltip"
                data-html
              >
                <span className={`tooltip ${baseClass}__tooltip-text`}>
                  {searchToolTipText}
                </span>
              </ReactTooltip>
            </div>
          )}
        </div>
      )}
      <div className={`${baseClass}__data-table-block`}>
        {/* No entities for this result. */}
        {(!isLoading && data.length === 0 && !isMultiColumnFilter) ||
        (searchQuery.length &&
          data.length === 0 &&
          !isMultiColumnFilter &&
          !isLoading) ? (
          <>
            <EmptyComponent pageIndex={pageIndex} />
            {pageIndex !== 0 && (
              <div className={`${baseClass}__empty-page`}>
                <div className={`${baseClass}__previous`}>
                  <Pagination
                    resultsOnCurrentPage={data.length}
                    currentPage={pageIndex}
                    resultsPerPage={pageSize}
                    onPaginationChange={onPaginationChange}
                  />
                </div>
              </div>
            )}
          </>
        ) : (
          <>
            {/* TODO: Fix this hacky solution to clientside search being 0 rendering emptycomponent but
            no longer accesses rows.length because DataTable is not rendered */}
            {!isLoading && clientFilterCount === 0 && !isMultiColumnFilter && (
              <EmptyComponent pageIndex={pageIndex} />
            )}
            <div
              className={
                isClientSideFilter && !isMultiColumnFilter
                  ? `client-result-count-${clientFilterCount}`
                  : ""
              }
            >
              <DataTable
                isLoading={isLoading}
                columns={columnConfigs}
                data={data}
                filters={filters}
                manualSortBy={manualSortBy}
                sortHeader={sortHeader}
                sortDirection={sortDirection}
                onSort={onSortChange}
                disableMultiRowSelect={disableMultiRowSelect}
                showMarkAllPages={showMarkAllPages}
                isAllPagesSelected={isAllPagesSelected}
                toggleAllPagesSelected={toggleAllPagesSelected}
                resultsTitle={resultsTitle}
                defaultPageSize={pageSize}
                defaultPageIndex={defaultPageIndex}
                primarySelectAction={primarySelectAction}
                secondarySelectActions={secondarySelectActions}
                onSelectSingleRow={onSelectSingleRow}
                onResultsCountChange={onResultsCountChange}
                isClientSidePagination={isClientSidePagination}
                onClientSidePaginationChange={onClientSidePaginationChange}
                isClientSideFilter={isClientSideFilter}
                disableHighlightOnHover={disableHighlightOnHover}
                searchQuery={searchQuery}
                searchQueryColumn={searchQueryColumn}
                selectedDropdownFilter={selectedDropdownFilter}
                renderFooter={renderFooter}
                renderPagination={
                  isClientSidePagination ? undefined : renderPagination
                }
                setExportRows={setExportRows}
              />
            </div>
          </>
        )}
      </div>
    </div>
  );
};

export default TableContainer;
