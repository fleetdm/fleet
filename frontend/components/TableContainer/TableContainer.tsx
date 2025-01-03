import React, { useState, useCallback, useRef, useEffect } from "react";
import classnames from "classnames";
import { Row } from "react-table";
import ReactTooltip from "react-tooltip";
import useDeepEffect from "hooks/useDeepEffect";
import { noop } from "lodash";

import SearchField from "components/forms/fields/SearchField";
// @ts-ignore
import Pagination from "components/Pagination";
import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";
import { COLORS } from "styles/var/colors";

import DataTable from "./DataTable/DataTable";
import { IActionButtonProps } from "./DataTable/ActionButton/ActionButton";

export interface ITableQueryData {
  pageIndex: number;
  pageSize: number;
  searchQuery: string;
  sortHeader: string;
  sortDirection: string;
}
interface IRowProps extends Row {
  original: {
    id?: number;
    os_version_id?: string; // Required for onSelectSingleRow of SoftwareOSTable.tsx
    cve?: string; // Required for onSelectSingleRow of SoftwareVulnerabilityTable.tsx
  };
}

interface ITableContainerProps<T = any> {
  columnConfigs: any; // TODO: Figure out type
  data: any; // TODO: Figure out type
  isLoading: boolean;
  manualSortBy?: boolean;
  defaultSortHeader?: string;
  defaultSortDirection?: string;
  defaultSearchQuery?: string;
  defaultPageIndex?: number;
  defaultSelectedRows?: Record<string, boolean>;
  /** Button visible above the table container next to search bar */
  actionButton?: IActionButtonProps;
  inputPlaceHolder?: string;
  disableActionButton?: boolean;
  disableMultiRowSelect?: boolean;
  /** resultsTitle used in DataTable for matching results text */
  resultsTitle?: string;
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
  searchToolTipText?: string;
  // TODO - consolidate this functionality within `filters`
  searchQueryColumn?: string;
  // TODO - consolidate this functionality within `filters`
  selectedDropdownFilter?: string;
  isClientSidePagination?: boolean;
  /** Used to set URL to correct path and include page query param */
  onClientSidePaginationChange?: (pageIndex: number) => void;
  /** Sets the table to filter the data on the client */
  isClientSideFilter?: boolean;
  /** isMultiColumnFilter is used to preserve the table headers
  in lieu of displaying the empty component when client-side filtering yields zero results */
  isMultiColumnFilter?: boolean;
  disableHighlightOnHover?: boolean;
  pageSize?: number;
  onQueryChange?:
    | ((queryData: ITableQueryData) => void)
    | ((queryData: ITableQueryData) => number);
  customControl?: () => JSX.Element | null;
  /** Filter button right of the search rendering alternative responsive design where search bar moves to new line but filter button remains inline with other table headers */
  customFiltersButton?: () => JSX.Element;
  stackControls?: boolean;
  onSelectSingleRow?: (value: Row | IRowProps) => void;
  /** This is called when you click on a row. This was added as `onSelectSingleRow`
   * only work if `disableMultiRowSelect` is also set to `true`. TODO: figure out
   * if we want to keep this
   */
  onClickRow?: (row: T) => void;
  /** Used if users can click the row and another child element does not have the same onClick functionality */
  keyboardSelectableRows?: boolean;
  /** Use for clientside filtering: Use key global for filtering on any column, or use column id as
   * key */
  filters?: Record<string, string | number | boolean>;
  renderCount?: () => JSX.Element | null;
  /** Optional help text to render on bottom-left of the table. Hidden when table is loading and no
   * rows of data are present. */
  renderTableHelpText?: () => JSX.Element | null;
  setExportRows?: (rows: Row[]) => void;
  /** Use for serverside filtering: Set to true when filters change in URL
   * bar and API call so TableContainer will reset its page state to 0  */
  resetPageIndex?: boolean;
  disableTableHeader?: boolean;
  /** Set to true to persist the row selections across table data filters */
  persistSelectedRows?: boolean;
  /** handler called when the  `clear selection` button is called */
  onClearSelection?: () => void;
}

const baseClass = "table-container";

const DEFAULT_PAGE_SIZE = 20;
const DEFAULT_PAGE_INDEX = 0;

const TableContainer = <T,>({
  columnConfigs,
  data,
  filters,
  isLoading,
  manualSortBy = false,
  defaultSearchQuery = "",
  defaultPageIndex = DEFAULT_PAGE_INDEX,
  defaultSortHeader = "name",
  defaultSortDirection = "asc",
  defaultSelectedRows,
  inputPlaceHolder = "Search",
  additionalQueries,
  resultsTitle,
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
  customFiltersButton,
  stackControls,
  onSelectSingleRow,
  onClickRow,
  keyboardSelectableRows,
  renderCount,
  renderTableHelpText,
  setExportRows,
  resetPageIndex,
  disableTableHeader,
  persistSelectedRows,
  onClearSelection = noop,
}: ITableContainerProps<T>) => {
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
    setSearchQuery(value.trim());
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

  const renderFilters = useCallback(() => {
    const opacity = isLoading ? { opacity: 0.4 } : { opacity: 1 };

    // New preferred pattern uses grid container/box to allow for more dynamic responsiveness
    // At low widths, search bar (3rd div of 4) moves above other 3 divs
    if (stackControls) {
      return (
        <div className="container">
          <div className="stackable-header">
            {renderCount && !disableCount && (
              <div
                className={`${baseClass}__results-count ${
                  stackControls ? "stack-table-controls" : ""
                }`}
                style={opacity}
              >
                {renderCount()}
              </div>
            )}
          </div>

          {actionButton && !actionButton.hideButton && (
            <div className="stackable-header">
              <Button
                disabled={disableActionButton}
                onClick={actionButton.onActionButtonClick}
                variant={actionButton.variant || "brand"}
                className={`${baseClass}__table-action-button`}
              >
                <>
                  {actionButton.buttonText}
                  {actionButton.iconSvg && <Icon name={actionButton.iconSvg} />}
                </>
              </Button>
            </div>
          )}
          <div className="stackable-header top-shift-header">
            {customControl ? customControl() : undefined}
            {searchable && !wideSearch && (
              <div className={`${baseClass}__search`}>
                <div
                  className={`${baseClass}__search-input`}
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
            {customFiltersButton && customFiltersButton()}
          </div>
        </div>
      );
    }
    return (
      <>
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
              {renderCount && !disableCount && (
                <div
                  className={`${baseClass}__results-count ${
                    stackControls ? "stack-table-controls" : ""
                  }`}
                  style={opacity}
                >
                  {renderCount()}
                </div>
              )}
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
      </>
    );
  }, [
    actionButton,
    customControl,
    customFiltersButton,
    disableActionButton,
    disableCount,
    disableTableHeader,
    inputPlaceHolder,
    isLoading,
    renderCount,
    searchQuery,
    searchToolTipText,
    searchable,
    stackControls,
    wideSearch,
  ]);

  return (
    <div className={wrapperClasses}>
      {renderFilters()}
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
                defaultSelectedRows={defaultSelectedRows}
                primarySelectAction={primarySelectAction}
                secondarySelectActions={secondarySelectActions}
                onSelectSingleRow={onSelectSingleRow}
                onClickRow={onClickRow}
                keyboardSelectableRows={keyboardSelectableRows}
                onResultsCountChange={setClientFilterCount}
                isClientSidePagination={isClientSidePagination}
                onClientSidePaginationChange={onClientSidePaginationChange}
                isClientSideFilter={isClientSideFilter}
                disableHighlightOnHover={disableHighlightOnHover}
                searchQuery={searchQuery}
                searchQueryColumn={searchQueryColumn}
                selectedDropdownFilter={selectedDropdownFilter}
                renderTableHelpText={renderTableHelpText}
                renderPagination={
                  isClientSidePagination ? undefined : renderPagination
                }
                setExportRows={setExportRows}
                onClearSelection={onClearSelection}
                persistSelectedRows={persistSelectedRows}
              />
            </div>
          </>
        )}
      </div>
    </div>
  );
};

export default TableContainer;
