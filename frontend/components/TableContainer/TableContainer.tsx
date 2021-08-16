import React, { useState, useEffect, useCallback, useRef } from "react";
import classnames from "classnames";
import { useAsyncDebounce } from "react-table";
import Button from "components/buttons/Button";
// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import InputField from "components/forms/fields/InputField";
// @ts-ignore
import Pagination from "components/Pagination";
// @ts-ignore
import scrollToTop from "utilities/scroll_to_top";

// @ts-ignore
import DataTable from "./DataTable/DataTable";
import TableContainerUtils from "./TableContainerUtils";
import { IActionButtonProps } from "./DataTable/ActionButton";

interface ITableQueryData {
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
  defaultSortHeader: string;
  defaultSortDirection: string;
  onActionButtonClick?: () => void;
  actionButtonText?: string;
  actionButtonIcon?: string;
  actionButtonVariant?: string;
  onQueryChange: (queryData: ITableQueryData) => void;
  inputPlaceHolder: string;
  disableActionButton?: boolean;
  resultsTitle: string;
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
  primarySelectActionButtonVariant?: string;
  primarySelectActionButtonIcon?: string;
  primarySelectActionButtonText?: string | ((targetIds: number[]) => string);
  onPrimarySelectActionClick?: (selectedItemIds: number[]) => void;
  secondarySelectActions?: IActionButtonProps[]; // TODO create table actions interface
  customControl?: () => JSX.Element;
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
  defaultSortHeader,
  defaultSortDirection,
  onActionButtonClick,
  inputPlaceHolder,
  additionalQueries,
  onQueryChange,
  resultsTitle,
  emptyComponent,
  className,
  disableActionButton,
  actionButtonText,
  actionButtonIcon,
  actionButtonVariant,
  showMarkAllPages,
  isAllPagesSelected,
  toggleAllPagesSelected,
  searchable,
  wideSearch,
  disablePagination,
  disableCount,
  primarySelectActionButtonVariant,
  primarySelectActionButtonIcon,
  primarySelectActionButtonText,
  onPrimarySelectActionClick,
  secondarySelectActions,
  customControl,
}: ITableContainerProps): JSX.Element => {
  const [searchQuery, setSearchQuery] = useState("");
  const [sortHeader, setSortHeader] = useState(defaultSortHeader || "");
  const [sortDirection, setSortDirection] = useState(
    defaultSortDirection || ""
  );
  const [pageSize] = useState(DEFAULT_PAGE_SIZE);
  const [pageIndex, setPageIndex] = useState(DEFAULT_PAGE_INDEX);

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
    scrollToTop();
  };

  // We use useRef to keep track of the previous searchQuery value. This allows us
  // to later compare this the the current value and debounce a change handler.
  const prevSearchQueryRef = useRef(searchQuery);
  const prevSearchQuery = prevSearchQueryRef.current;
  const debounceOnQueryChange = useAsyncDebounce(
    (queryData: ITableQueryData) => {
      onQueryChange(queryData);
    },
    DEBOUNCE_QUERY_DELAY
  );

  // When any of our query params change, or if any additionalQueries change, we want to fire off
  // the parent components handler function with this updated query data. There is logic in here to check
  // different types of query updates, as we handle some of them differently than others.
  useEffect(() => {
    const queryData = {
      searchQuery,
      sortHeader,
      sortDirection,
      pageSize,
      pageIndex,
    };
    // Something besides the pageIndex has changed; we want to set it back to 0.
    if (!hasPageIndexChangedRef.current) {
      const updateQueryData = {
        ...queryData,
        pageIndex: 0,
      };
      // searchQuery has changed; we want to debounce calling the handler so the
      // user can finish typing.
      if (searchQuery !== prevSearchQuery) {
        debounceOnQueryChange(updateQueryData);
      } else {
        onQueryChange(updateQueryData);
      }
      setPageIndex(0);
    } else {
      onQueryChange(queryData);
    }

    hasPageIndexChangedRef.current = false;
  }, [
    searchQuery,
    sortHeader,
    sortDirection,
    pageSize,
    pageIndex,
    additionalQueries,
    onQueryChange,
    debounceOnQueryChange,
    prevSearchQuery,
  ]);

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
        {data && data.length && !disableCount ? (
          <p className={`${baseClass}__results-count`}>
            {TableContainerUtils.generateResultsCountText(
              resultsTitle,
              pageIndex,
              pageSize,
              data.length
            )}
          </p>
        ) : (
          <p />
        )}
        <div className={`${baseClass}__table-controls`}>
          {actionButtonText && (
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
            <div className={`${baseClass}__search-input`}>
              <InputField
                placeholder={inputPlaceHolder}
                name="searchQuery"
                onChange={onSearchQueryChange}
                value={searchQuery}
                inputWrapperClass={`${baseClass}__input-wrapper`}
              />
            </div>
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
            <DataTable
              isLoading={isLoading}
              columns={columns}
              data={data}
              manualSortBy={manualSortBy}
              sortHeader={sortHeader}
              sortDirection={sortDirection}
              onSort={onSortChange}
              showMarkAllPages={showMarkAllPages}
              isAllPagesSelected={isAllPagesSelected}
              toggleAllPagesSelected={toggleAllPagesSelected}
              resultsTitle={resultsTitle}
              defaultPageSize={DEFAULT_PAGE_SIZE}
              primarySelectActionButtonVariant={
                primarySelectActionButtonVariant
              }
              primarySelectActionButtonIcon={primarySelectActionButtonIcon}
              primarySelectActionButtonText={primarySelectActionButtonText}
              onPrimarySelectActionClick={onPrimarySelectActionClick}
              secondarySelectActions={secondarySelectActions}
            />
            {!disablePagination && (
              <Pagination
                resultsOnCurrentPage={data.length}
                currentPage={pageIndex}
                resultsPerPage={pageSize}
                onPaginationChange={onPaginationChange}
              />
            )}
          </>
        )}
      </div>
    </div>
  );
};

export default TableContainer;
