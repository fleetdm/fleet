import React, { useState, useCallback, useRef, useEffect } from "react";
import classnames from "classnames";
import { Row } from "react-table";
import ReactTooltip from "react-tooltip";
import useDeepEffect from "hooks/useDeepEffect";
import { noop } from "lodash";

import SearchField from "components/forms/fields/SearchField";
import Pagination from "components/Pagination";
import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";
import TooltipWrapper from "components/TooltipWrapper";

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

// TODO - there are 2 `IActionButtonProps` - reconcile
interface ITableContainerActionButtonProps extends IActionButtonProps {
  disabledTooltipContent?: React.ReactNode;
}

interface ITableContainerProps<T = any> {
  columnConfigs: any; // TODO: Figure out type
  data: any; // TODO: Figure out type
  isLoading: boolean;
  manualSortBy?: boolean;
  defaultSortHeader?: string;
  defaultSortDirection?: string;
  defaultSearchQuery?: string;
  /**  Used for client-side filtering with a search query controlled outside TableContainer */
  searchQuery?: string;
  /**  When page index is externally managed like from the URL, this prop must be set to control currentPageIndex */
  pageIndex?: number;
  defaultSelectedRows?: Record<string, boolean>;
  /** Button visible above the table container next to search bar */
  actionButton?: ITableContainerActionButtonProps;
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
  /**
   * Disables the "Next" button when the last page contains exactly the page size items.
   * This is determined using either the API's `meta.has_next_page` response
   * or by calculating `isLastPage` in the frontend.
   */
  disableNextPage?: boolean;
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
  disableTableHeader?: boolean;
  /** Set to true to persist the row selections across table data filters */
  persistSelectedRows?: boolean;
  /** Set to `true` to not display the footer section of the table */
  hideFooter?: boolean;
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
  searchQuery: controlledSearchQuery,
  pageIndex = DEFAULT_PAGE_INDEX,
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
  hideFooter,
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
  disableTableHeader,
  persistSelectedRows,
  onClearSelection = noop,
}: ITableContainerProps<T>) => {
  const isControlledSearchQuery = controlledSearchQuery !== undefined;
  const [searchQuery, setSearchQuery] = useState(defaultSearchQuery);
  const [sortHeader, setSortHeader] = useState(defaultSortHeader || "");
  const [sortDirection, setSortDirection] = useState(
    defaultSortDirection || ""
  );
  const [currentPageIndex, setCurrentPageIndex] = useState<number>(pageIndex);
  const [clientFilterCount, setClientFilterCount] = useState<number>();

  // If using a clientside search query outside of TableContainer,
  // we need to set the searchQuery state to the controlledSearchQuery prop anytime it changes
  useEffect(() => {
    if (isControlledSearchQuery) {
      setSearchQuery(controlledSearchQuery);
    }
  }, [controlledSearchQuery, isControlledSearchQuery]);

  // Client side pagination is being overridden to previous page without this
  useEffect(() => {
    if (isClientSidePagination && currentPageIndex !== DEFAULT_PAGE_INDEX) {
      setCurrentPageIndex(DEFAULT_PAGE_INDEX);
    }
  }, [currentPageIndex, isClientSidePagination]);

  // pageIndex must update currentPageIndex anytime it's changed or else it causes bugs
  // e.g. bug of filter dd not reverting table to page 0
  useEffect(() => {
    if (!isClientSidePagination) {
      setCurrentPageIndex(pageIndex);
    }
  }, [pageIndex, isClientSidePagination]);

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

  const onPaginationChange = useCallback(
    (newPage: number) => {
      if (!isClientSidePagination) {
        setCurrentPageIndex(newPage);
      }
    },
    [isClientSidePagination]
  );

  useDeepEffect(() => {
    if (!onQueryChange) {
      return;
    }

    const queryData = {
      searchQuery,
      sortHeader,
      sortDirection,
      pageSize,
      pageIndex: currentPageIndex,
    };

    if (prevPageIndex.current === currentPageIndex) {
      setCurrentPageIndex(0);
    }

    // NOTE: used to reset page number to 0 when modifying filters
    const newPageIndex = onQueryChange(queryData);
    if (newPageIndex === 0) {
      setCurrentPageIndex(0);
    }

    prevPageIndex.current = currentPageIndex;
  }, [
    searchQuery,
    sortHeader,
    sortDirection,
    pageSize,
    currentPageIndex,
    additionalQueries,
  ]);

  /** This is server side pagination. Clientside pagination is handled in
   * data table using react-table builtins */
  const renderServersidePagination = useCallback(() => {
    if (disablePagination || isClientSidePagination) {
      return null;
    }
    return (
      <Pagination
        disablePrev={pageIndex === 0}
        disableNext={disableNextPage || data.length < pageSize}
        onPrevPage={() => onPaginationChange(pageIndex - 1)}
        onNextPage={() => onPaginationChange(pageIndex + 1)}
        hidePagination={
          (disableNextPage || data.length < pageSize) && pageIndex === 0
        }
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

  const renderFilterActionButton = () => {
    // always !!actionButton here, this is for type checker
    if (actionButton) {
      const button = (
        <Button
          disabled={
            !!actionButton.disabledTooltipContent || disableActionButton
          }
          onClick={actionButton.onClick}
          variant={actionButton.variant || "default"}
          className={`${baseClass}__table-action-button`}
        >
          <>
            {actionButton.buttonText}
            {actionButton.iconSvg && <Icon name={actionButton.iconSvg} />}
          </>
        </Button>
      );
      return actionButton.disabledTooltipContent ? (
        <TooltipWrapper
          tipContent={actionButton.disabledTooltipContent}
          position="top"
          underline={false}
          showArrow
          tipOffset={8}
        >
          {button}
        </TooltipWrapper>
      ) : (
        button
      );
    }
  };

  const renderFilters = useCallback(() => {
    const opacity = isLoading ? { opacity: 0.4 } : { opacity: 1 };

    // New preferred pattern uses grid container/box to allow for more dynamic responsiveness
    // At low widths, right header stacks on top of left header
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
            <div className="stackable-header">{renderFilterActionButton()}</div>
          )}
          <div className="stackable-header top-shift-header">
            {customControl && customControl()}
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
                {actionButton &&
                  !actionButton.hideButton &&
                  renderFilterActionButton()}
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
            <EmptyComponent pageIndex={currentPageIndex} />
            {/* This UI only shows if a user navigates to a table page with a URL page param that is outside the # of pages available */}
            {currentPageIndex !== 0 && (
              <div className={`${baseClass}__empty-page`}>
                <div className={`${baseClass}__previous-button`}>
                  <Pagination
                    disableNext
                    onNextPage={() => onPaginationChange(currentPageIndex + 1)}
                    onPrevPage={() => onPaginationChange(currentPageIndex - 1)}
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
              <EmptyComponent pageIndex={currentPageIndex} />
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
                defaultPageIndex={pageIndex}
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
                  isClientSidePagination
                    ? undefined
                    : renderServersidePagination
                }
                setExportRows={setExportRows}
                onClearSelection={onClearSelection}
                persistSelectedRows={persistSelectedRows}
                hideFooter={hideFooter}
              />
            </div>
          </>
        )}
      </div>
    </div>
  );
};

export default TableContainer;
