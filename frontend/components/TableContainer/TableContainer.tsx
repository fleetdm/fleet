import React, { useState, useEffect, useCallback, useRef } from 'react';
import classnames from 'classnames';
import { useAsyncDebounce } from 'react-table';

import Button from 'components/buttons/Button';
// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import InputField from 'components/forms/fields/InputField';
// @ts-ignore
import KolideIcon from 'components/icons/KolideIcon';
// @ts-ignore
import Pagination from 'components/Pagination';
// @ts-ignore
import scrollToTop from 'utilities/scroll_to_top';

// @ts-ignore
import DataTable from './DataTable/DataTable';


import TableContainerUtils from './TableContainerUtils';

interface ITableQueryData {
  searchQuery: string;
  sortHeader: string;
  sortDirection: string;
  pageSize: number;
  pageIndex: number;
}

interface ITableContainerProps<T, U> {
  columns: T[];
  data: U[];
  isLoading: boolean;
  defaultSortHeader: string;
  defaultSortDirection: string;
  onActionButtonClick: () => void;
  actionButtonText: string;
  onQueryChange: (queryData: ITableQueryData) => void;
  inputPlaceHolder: string;
  includesTableActionButton?: boolean; // will be used later to conditionally show button
  disableActionButton?: boolean;
  resultsTitle?: string;
  additionalQueries?: string;
  emptyComponent: React.ElementType;
  className?: string;
}

const baseClass = 'table-container';

const DEFAULT_PAGE_SIZE = 100;
const DEFAULT_PAGE_INDEX = 0;
const DEBOUNCE_QUERY_DELAY = 300;

const TableContainer = <T, U>(props: ITableContainerProps<T, U>): JSX.Element => {
  const {
    columns,
    data,
    isLoading,
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
  } = props;

  const [searchQuery, setSearchQuery] = useState('');
  const [sortHeader, setSortHeader] = useState(defaultSortHeader || '');
  const [sortDirection, setSortDirection] = useState(defaultSortDirection || '');
  const [pageSize] = useState(DEFAULT_PAGE_SIZE);
  const [pageIndex, setPageIndex] = useState(DEFAULT_PAGE_INDEX);

  const wrapperClasses = classnames(baseClass, className);

  const EmptyComponent = emptyComponent;

  const onSortChange = useCallback((id?:string, isDesc?: boolean) => {
    if (id === undefined) {
      setSortHeader('');
      setSortDirection('');
    } else {
      setSortHeader(id);
      const direction = isDesc ? 'desc' : 'asc';
      setSortDirection(direction);
    }
  }, [setSortHeader, setSortDirection]);


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
  const debounceOnQueryChange = useAsyncDebounce((queryData: ITableQueryData) => {
    onQueryChange(queryData);
  }, DEBOUNCE_QUERY_DELAY);

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
  }, [searchQuery, sortHeader, sortDirection, pageSize, pageIndex, additionalQueries, onQueryChange]);

  return (
    <div className={wrapperClasses}>
      {/* TODO: find a way to move these controls into the table component */}
      <div className={`${baseClass}__header`}>
        { data && data.length ?
          <p className={`${baseClass}__results-count`}>
            {TableContainerUtils.generateResultsCountText(resultsTitle, pageIndex, pageSize, data.length)}
          </p> :
          <p />
        }
        <div className={`${baseClass}__table-controls`}>
          <Button
            disabled={disableActionButton}
            onClick={onActionButtonClick}
            variant="unstyled"
            className={`${baseClass}__table-action-button`}
          >
            {actionButtonText}
          </Button>
          <div className={`${baseClass}__search-input`}>
            <InputField
              placeholder={inputPlaceHolder}
              name="searchQuery"
              onChange={onSearchQueryChange}
              value={searchQuery}
              inputWrapperClass={`${baseClass}__input-wrapper`}
            />
            <KolideIcon name="search" />
          </div>
        </div>
      </div>
      <div className={`${baseClass}__data-table-container`}>
        {/* No entities for this result. */}
        {!isLoading && data.length === 0 ?
          <EmptyComponent /> :
          <>
            <DataTable
              isLoading={isLoading}
              columns={columns}
              data={data}
              sortHeader={sortHeader}
              sortDirection={sortDirection}
              onSort={onSortChange}
              resultsName={'hosts'}
            />
            <Pagination
              resultsOnCurrentPage={data.length}
              currentPage={pageIndex}
              resultsPerPage={pageSize}
              onPaginationChange={onPaginationChange}
            />
          </>

        }
      </div>
    </div>
  );
};

export default TableContainer;
