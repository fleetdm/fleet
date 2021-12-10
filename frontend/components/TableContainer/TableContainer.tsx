import React, { useState, useCallback, useRef } from "react";
import classnames from "classnames";
import { Row, useAsyncDebounce } from "react-table";

// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import InputField from "components/forms/fields/InputField"; // @ts-ignore
import Pagination from "components/Pagination";
import Button from "components/buttons/Button";
import { ButtonVariant } from "components/buttons/Button/Button"; // @ts-ignore
import { useDeepEffect } from "utilities/hooks";
import ReactTooltip from "react-tooltip";

// @ts-ignore
import DataTable from "./DataTable/DataTable";
import TableContainerUtils from "./TableContainerUtils";
import { IActionButtonProps } from "./DataTable/ActionButton";

export interface ITableSearchData {
  searchQuery: string;
  sortHeader: string;
  sortDirection: string;
  pageSize: number;
  pageIndex: number;
}

interface ITableContainerProps {
  columns: any; // TODO: Figure out type
  data: any; // TODO: Figure out type
  isLoading: boolean;
  manualSortBy?: boolean;
  defaultSortHeader?: string;
  defaultSortDirection?: string;
  onActionButtonClick?: () => void;
  actionButtonText?: string;
  actionButtonIcon?: string;
  actionButtonVariant?: ButtonVariant;
  hideActionButton?: boolean;
  onQueryChange?: (queryData: ITableSearchData) => void;
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
  disableCount?: boolean;
  primarySelectActionButtonVariant?: ButtonVariant;
  primarySelectActionButtonIcon?: string;
  primarySelectActionButtonText?: string | ((targetIds: number[]) => string);
  onPrimarySelectActionClick?: (selectedItemIds: number[]) => void;
  secondarySelectActions?: IActionButtonProps[]; // TODO create table actions interface
  customControl?: () => JSX.Element;
  onSelectSingleRow?: (value: Row) => void;
  filteredCount?: number;
  searchToolTipText?: string;
  searchQueryColumn?: string;
  selectedDropdownFilter?: string;
  isClientSidePagination?: boolean;
  isClientSideFilter?: boolean;
  isClientSideSearch?: boolean;
  highlightOnHover?: boolean;
  pageSize?: number;
}

const baseClass = "table-container";

const DEFAULT_PAGE_SIZE = 100;
const DEFAULT_PAGE_INDEX = 0;
const DEBOUNCE_QUERY_DELAY = 300;

const TableContainer = ({
  columns,
  data,
  isLoading,
  manualSortBy = false,
  defaultSortHeader = "name",
  defaultSortDirection = "asc",
  onActionButtonClick,
  inputPlaceHolder = "Search",
  additionalQueries,
  onQueryChange,
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
  disableCount,
  primarySelectActionButtonVariant = "brand",
  primarySelectActionButtonIcon,
  primarySelectActionButtonText,
  onPrimarySelectActionClick,
  secondarySelectActions,
  customControl,
  onSelectSingleRow,
  filteredCount,
  searchToolTipText,
  isClientSidePagination,
  isClientSideFilter,
  isClientSideSearch,
  highlightOnHover,
  pageSize = DEFAULT_PAGE_SIZE,
  selectedDropdownFilter,
  searchQueryColumn,
}: ITableContainerProps): JSX.Element => {
  const [searchQuery, setSearchQuery] = useState("");
  const [sortHeader, setSortHeader] = useState(defaultSortHeader || "");
  const [sortDirection, setSortDirection] = useState(
    defaultSortDirection || ""
  );
  const [pageIndex, setPageIndex] = useState<number>(DEFAULT_PAGE_INDEX);
  const [clientFilterCount, setClientFilterCount] = useState<number>();

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
  const onPaginationChange = (newPage: number) => {
    setPageIndex(newPage);
    hasPageIndexChangedRef.current = true;
  };

  const onResultsCountChange = (resultsCount: number) => {
    setClientFilterCount(resultsCount);
  };

  // We use useRef to keep track of the previous searchQuery value. This allows us
  // to later compare this the the current value and debounce a change handler.
  const prevSearchQueryRef = useRef(searchQuery);
  const prevSearchQuery = prevSearchQueryRef.current;
  const debounceOnQueryChange = useAsyncDebounce(
    (queryData: ITableSearchData) => {
      onQueryChange && onQueryChange(queryData);
    },
    DEBOUNCE_QUERY_DELAY
  );

  // When any of our query params change, or if any additionalQueries change, we want to fire off
  // the parent components handler function with this updated query data. There is logic in here to check
  // different types of query updates, as we handle some of them differently than others.
  useDeepEffect(() => {
    const queryData = {
      searchQuery,
      sortHeader,
      sortDirection,
      pageSize,
      pageIndex,
    };

    // Something besides the pageIndex has changed; we want to set it back to 0.
    if (onQueryChange) {
      if (!hasPageIndexChangedRef.current && !isClientSideSearch) {
        const updateQueryData = {
          ...queryData,
          pageIndex: 0,
        };
        if (!isClientSideFilter) {
          // searchQuery has changed; we want to debounce calling the handler so the
          // user can finish typing.
          if (searchQuery !== prevSearchQuery) {
            debounceOnQueryChange(updateQueryData);
          } else {
            onQueryChange(updateQueryData);
          }
          setPageIndex(0);
        } else {
          onQueryChange(updateQueryData);
        }
      } else if (!isClientSideFilter) {
        onQueryChange(queryData);
      }

      hasPageIndexChangedRef.current = false;
    }
  }, [
    searchQuery,
    sortHeader,
    sortDirection,
    pageSize,
    pageIndex,
    additionalQueries,
    prevSearchQuery,
  ]);

  const displayCount = useCallback((): number => {
    if (typeof filteredCount === "number") {
      return filteredCount;
    } else if (typeof clientFilterCount === "number") {
      return clientFilterCount;
    }
    return data.length;
  }, [filteredCount, clientFilterCount, data]);

  return (
    <div className={wrapperClasses}>
      {wideSearch && searchable && (
        <div className={`${baseClass}__search-input wide-search`}>
          <InputField
            placeholder={inputPlaceHolder}
            name="searchQuery"
            onChange={onSearchQueryChange}
            value={searchQuery}
            inputWrapperClass={`${baseClass}__input-wrapper`}
          />
        </div>
      )}
      <div className={`${baseClass}__header`}>
        {data && displayCount() && !disableCount ? (
          <p className={`${baseClass}__results-count`}>
            {TableContainerUtils.generateResultsCountText(
              resultsTitle,
              displayCount()
            )}
            {resultsHtml}
          </p>
        ) : (
          <p />
        )}
        <div className={`${baseClass}__table-controls`}>
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
          {/* Render search bar only if not empty component */}
          {searchable && !wideSearch && (
            <>
              <div
                className={`${baseClass}__search-input`}
                data-tip
                data-for="search-tooltip"
                data-tip-disable={!searchToolTipText}
              >
                <InputField
                  placeholder={inputPlaceHolder}
                  name="searchQuery"
                  onChange={onSearchQueryChange}
                  value={searchQuery}
                  inputWrapperClass={`${baseClass}__input-wrapper`}
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
      <div className={`${baseClass}__data-table-container`}>
        {/* No entities for this result. */}
        {!isLoading && data.length === 0 ? (
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
              />
              {!disablePagination && !isClientSidePagination && (
                <Pagination
                  resultsOnCurrentPage={data.length}
                  currentPage={pageIndex}
                  resultsPerPage={pageSize}
                  onPaginationChange={onPaginationChange}
                />
              )}
            </div>
          </>
        )}
      </div>
    </div>
  );
};

export default TableContainer;
