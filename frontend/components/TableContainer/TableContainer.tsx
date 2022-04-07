import React, { useState, useCallback, useRef } from "react";
import classnames from "classnames";
import { Row } from "react-table";

// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import SearchField from "components/forms/fields/SearchField";
// @ts-ignore
import Pagination from "components/Pagination";
import Button from "components/buttons/Button";
import { ButtonVariant } from "components/buttons/Button/Button";
// @ts-ignore
import { useDeepEffect } from "utilities/hooks";
import ReactTooltip from "react-tooltip";

// @ts-ignore
import DataTable from "./DataTable/DataTable";
import TableContainerUtils from "./TableContainerUtils";
import { IActionButtonProps } from "./DataTable/ActionButton";

export interface ITableQueryData {
  pageIndex: number;
  pageSize: number;
  searchQuery: string;
  sortHeader: string;
  sortDirection: string;
}

interface ITableContainerProps {
  columns: any; // TODO: Figure out type
  data: any; // TODO: Figure out type
  isLoading: boolean;
  manualSortBy?: boolean;
  defaultSortHeader?: string;
  defaultSortDirection?: string;
  actionButtonText?: string;
  actionButtonIcon?: string;
  actionButtonVariant?: ButtonVariant;
  hideActionButton?: boolean;
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
  primarySelectActionButtonVariant?: ButtonVariant;
  primarySelectActionButtonIcon?: string;
  primarySelectActionButtonText?: string | ((targetIds: number[]) => string);
  secondarySelectActions?: IActionButtonProps[]; // TODO create table actions interface
  filteredCount?: number;
  searchToolTipText?: string;
  searchQueryColumn?: string;
  selectedDropdownFilter?: string;
  isClientSidePagination?: boolean;
  isClientSideFilter?: boolean;
  highlightOnHover?: boolean;
  pageSize?: number;
  onActionButtonClick?: () => void;
  onQueryChange?: (queryData: ITableQueryData) => void;
  onPrimarySelectActionClick?: (selectedItemIds: number[]) => void;
  customControl?: () => JSX.Element;
  stackControls?: boolean;
  onSelectSingleRow?: (value: Row) => void;
  filters?: Record<string, string | number | boolean>;
  renderCount?: () => JSX.Element | null;
  renderFooter?: () => JSX.Element | null;
}

const baseClass = "table-container";

const DEFAULT_PAGE_SIZE = 100;
const DEFAULT_PAGE_INDEX = 0;

const TableContainer = ({
  columns,
  data,
  filters,
  isLoading,
  manualSortBy = false,
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
  actionButtonText,
  actionButtonIcon,
  actionButtonVariant = "brand",
  hideActionButton,
  showMarkAllPages,
  isAllPagesSelected,
  toggleAllPagesSelected,
  searchable,
  wideSearch,
  disablePagination,
  disableNextPage,
  disableCount,
  primarySelectActionButtonVariant = "brand",
  primarySelectActionButtonIcon,
  primarySelectActionButtonText,
  secondarySelectActions,
  filteredCount,
  searchToolTipText,
  isClientSidePagination,
  isClientSideFilter,
  highlightOnHover,
  pageSize = DEFAULT_PAGE_SIZE,
  selectedDropdownFilter,
  searchQueryColumn,
  onActionButtonClick,
  onQueryChange,
  onPrimarySelectActionClick,
  customControl,
  stackControls,
  onSelectSingleRow,
  renderCount,
  renderFooter,
}: ITableContainerProps): JSX.Element => {
  const [searchQuery, setSearchQuery] = useState("");
  const [sortHeader, setSortHeader] = useState(defaultSortHeader || "");
  const [sortDirection, setSortDirection] = useState(
    defaultSortDirection || ""
  );
  const [pageIndex, setPageIndex] = useState<number>(DEFAULT_PAGE_INDEX);
  const [clientFilterCount, setClientFilterCount] = useState<number>();

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
      setPageIndex(newPage);
      hasPageIndexChangedRef.current = true;
    },
    [hasPageIndexChangedRef]
  );

  const onResultsCountChange = (resultsCount: number) => {
    setClientFilterCount(resultsCount);
  };

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

    onQueryChange(queryData);

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
    return data.length;
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
            onChange={onSearchQueryChange}
          />
        </div>
      )}
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
            {!renderCount && data && displayCount() && !disableCount ? (
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
          <span className={"controls"}>
            {!hideActionButton && actionButtonText && (
              <Button
                disabled={disableActionButton}
                onClick={onActionButtonClick}
                variant={actionButtonVariant}
                className={`${baseClass}__table-action-button`}
              >
                <>
                  {actionButtonText}
                  {actionButtonIcon && (
                    <img
                      src={actionButtonIcon}
                      alt={`${actionButtonText} icon`}
                    />
                  )}
                </>
              </Button>
            )}
            {customControl && customControl()}
          </span>
        </div>

        <div className={`${baseClass}__search`}>
          {/* Render search bar only if not empty component */}
          {searchable && !wideSearch && (
            <>
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
                  onChange={onSearchQueryChange}
                />
              </div>
              <ReactTooltip
                place="top"
                type="dark"
                effect="solid"
                backgroundColor="#3e4771"
                id="search-tooltip"
                data-html
              >
                <span className={`tooltip ${baseClass}__tooltip-text`}>
                  {searchToolTipText}
                </span>
              </ReactTooltip>
            </>
          )}
        </div>
      </div>
      <div className={`${baseClass}__data-table-block`}>
        {/* No entities for this result. */}
        {(!isLoading && data.length === 0) ||
        (searchQuery.length && data.length === 0) ? (
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
            {clientFilterCount === 0 && (
              <EmptyComponent pageIndex={pageIndex} />
            )}
            <div
              className={
                isClientSideFilter
                  ? `client-result-count-${clientFilterCount}`
                  : ""
              }
            >
              <DataTable
                isLoading={isLoading}
                columns={columns}
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
                primarySelectActionButtonVariant={
                  primarySelectActionButtonVariant
                }
                primarySelectActionButtonIcon={primarySelectActionButtonIcon}
                primarySelectActionButtonText={primarySelectActionButtonText}
                onPrimarySelectActionClick={onPrimarySelectActionClick}
                secondarySelectActions={secondarySelectActions}
                onSelectSingleRow={onSelectSingleRow}
                onResultsCountChange={onResultsCountChange}
                isClientSidePagination={isClientSidePagination}
                isClientSideFilter={isClientSideFilter}
                highlightOnHover={highlightOnHover}
                searchQuery={searchQuery}
                searchQueryColumn={searchQueryColumn}
                selectedDropdownFilter={selectedDropdownFilter}
                renderFooter={renderFooter}
                renderPagination={renderPagination}
              />
            </div>
          </>
        )}
      </div>
    </div>
  );
};

export default TableContainer;
