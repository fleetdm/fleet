import React, { useState, useEffect } from 'react';
import classnames from 'classnames';

import Button from 'components/buttons/Button';
// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import InputField from 'components/forms/fields/InputField';
// @ts-ignore
import KolideIcon from 'components/icons/KolideIcon';
// @ts-ignore
import Pagination from 'components/Pagination';
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
  includesTableAction: boolean;
  onTableActionClick: () => void;
  onQueryChange: (queryData: ITableQueryData) => void;
  resultsTitle?: string;
  additionalQueries?: string;
  className?: string;
}

const baseClass = 'table-container';

const DEFAULT_PAGE_SIZE = 100;
const DEFAULT_PAGE_INDEX = 0;

const TableContainer = <T, U>(props: ITableContainerProps<T, U>): JSX.Element => {
  const {
    columns,
    data,
    isLoading,
    defaultSortHeader,
    defaultSortDirection,
    onTableActionClick,
    includesTableAction,
    additionalQueries,
    onQueryChange,
    resultsTitle,
    className,
  } = props;

  const [searchQuery, setSearchQuery] = useState('');
  const [sortHeader, setSortHeader] = useState(defaultSortHeader || '');
  const [sortDirection, setSortDirection] = useState(defaultSortDirection || '');
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [pageIndex, setPageIndex] = useState(DEFAULT_PAGE_INDEX);

  const wrapperClasses = classnames(baseClass, className);

  // When any of our query params change, or if any additionalQueries change, we want to fire off
  // the parent components handler function with this updated query data.
  useEffect(() => {
    onQueryChange({
      searchQuery,
      sortHeader,
      sortDirection,
      pageSize,
      pageIndex,
    });
  }, [searchQuery, sortHeader, sortDirection, pageSize, pageIndex, additionalQueries, onQueryChange]);

  return (
    <div className={wrapperClasses}>
      {/* TODO: find a way to move these controls into the table component */}
      <div className={`${baseClass}__table-controls`}>
        { data && data.length ?
          <p className={`${baseClass}__results-count`}>
            {TableContainerUtils.generateResultsCountText(resultsTitle, pageIndex, pageSize, data.length)}
          </p> :
          null
        }
        <Button
          onClick={onTableActionClick}
          variant="unstyled"
          className={`${baseClass}__edit-columns-button`}
        >
          Edit columns
        </Button>
        <div className={`${baseClass}__search-input`}>
          <InputField
            placeholder="Search hostname, UUID, serial number, or IPv4"
            name=""
            onChange={setSearchQuery}
            value={searchQuery}
            inputWrapperClass={`${baseClass}__input-wrapper`}
          />
          <KolideIcon name="search" />
        </div>
      </div>
      <div className={`${baseClass}__data-table-container`}>
        {/* No entities for this result. */}
        {!isLoading && data.length === 0 ?
          <p>NO RESULTS</p> :
          <>
            <DataTable
              columns={columns}
              data={data}
              searchQuery={searchQuery}
              pageSize={pageSize}
              defaultSortHeader={sortHeader}
              resultsName={'hosts'}
              emptyComponent={<p>Empty</p>}
            />
            <Pagination
              resultsOnCurrentPage={data.length}
              currentPage={pageIndex}
              resultsPerPage={pageSize}
              onPaginationChange={setPageIndex}
            />
          </>
        // const NoResultsComponent = emptyComponent;
        // return <NoResultsComponent />;
        }
      </div>

    </div>
  );
};

export default TableContainer;
