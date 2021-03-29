import React, { useMemo, useEffect, useRef, useCallback, useState } from 'react';
import PropTypes from 'prop-types';
import { useTable, useGlobalFilter, useSortBy, useAsyncDebounce } from 'react-table';

import Spinner from 'components/loaders/Spinner';
import scrollToTop from 'utilities/scroll_to_top';

const DEFAULT_PAGE_INDEX = 0;
const DEBOUNCE_QUERY_DELAY = 300;
const DEFAULT_RESULTS_NAME = 'results';

const baseClass = 'data-table-container';

const generateResultsCountText = (name = DEFAULT_RESULTS_NAME, pageIndex, itemsPerPage, resultsCount) => {
  if (itemsPerPage === resultsCount) return `${itemsPerPage}+ ${name}`;

  if (pageIndex !== 0 && (resultsCount <= itemsPerPage)) return `${itemsPerPage}+ ${name}`;

  return `${resultsCount} ${name}`;
};

// This data table uses react-table for implementation. The relevant documentation of the library
// can be found here https://react-table.tanstack.com/docs/api/useTable
const DataTable = (props) => {
  const {
    tableColumns,
    tableData,
    isLoading,
    defaultSortHeader,
  } = props;

  // This variable is used to keep the react-table state persistent across server calls for new data.
  // You can read more about this here technique here:
  // https://react-table.tanstack.com/docs/faq#how-do-i-stop-my-table-state-from-automatically-resetting-when-my-data-changes
  // const skipPageResetRef = useRef();

  const columns = useMemo(() => {
    return tableColumns;
  }, [tableColumns]);

  // The table data needs to be ordered by the order we received from the API.
  const data = useMemo(() => {
    return tableData;
  }, [tableData]);

  const {
    headerGroups,
    rows,
    prepareRow,
    setGlobalFilter,
    state: tableState,
  } = useTable(
    {
      columns,
      data,
      initialState: {
        sortBy: [{ id: defaultSortHeader, desc: true }],
      },
      disableMultiSort: true,
      manualGlobalFilter: true,
      manualSortBy: true,
      // autoResetSortBy: !skipPageResetRef.current,
      // autoResetGlobalFilter: !skipPageResetRef.current,
    },
    useGlobalFilter,
    useSortBy,
  );
  const { globalFilter, sortBy } = tableState;

  const debouncedGlobalFilter = useAsyncDebounce((value) => {
    // skipPageResetRef.current = true;
    setGlobalFilter(value || undefined);
  }, DEBOUNCE_QUERY_DELAY);

  return (
    <div className={baseClass}>
      <div className={'data-table data-table__wrapper'}>
        {isLoading &&
          <div className={'loading-overlay'}>
            <Spinner />
          </div>
        }
        <table className={'data-table__table'}>
          <thead>
            {headerGroups.map(headerGroup => (
              <tr {...headerGroup.getHeaderGroupProps()}>
                {headerGroup.headers.map(column => (
                  <th {...column.getHeaderProps(column.getSortByToggleProps())}>
                    {column.render('Header')}
                  </th>
                ))}
              </tr>
            ))}
          </thead>
          <tbody>
            {rows.map((row) => {
              prepareRow(row);
              return (
                <tr {...row.getRowProps()}>
                  {row.cells.map((cell) => {
                    return (
                      <td {...cell.getCellProps()}>
                        {cell.render('Cell')}
                      </td>
                    );
                  })}
                </tr>
              );
            })
            }
          </tbody>
        </table>
      </div>
    </div>
  );
};

DataTable.propTypes = {
  tableColumns: PropTypes.arrayOf(PropTypes.object), // TODO: create proper interface for this
  tableData: PropTypes.arrayOf(PropTypes.object), // TODO: create proper interface for this
  isLoading: PropTypes.bool,
  defaultSortHeader: PropTypes.string,
  resultsName: PropTypes.string,
  fetchDataAction: PropTypes.func,
  emptyComponent: PropTypes.element,
};

export default DataTable;
