import React, { useMemo, useEffect } from 'react';
import PropTypes from 'prop-types';
import { useTable, useSortBy } from 'react-table';

import Spinner from 'components/loaders/Spinner';

const baseClass = 'data-table-container';

// This data table uses react-table for implementation. The relevant documentation of the library
// can be found here https://react-table.tanstack.com/docs/api/useTable
const DataTable = (props) => {
  const {
    columns: tableColumns,
    data: tableData,
    isLoading,
    sortHeader,
    sortDirection,
    onSort,
  } = props;

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
    state: tableState,
  } = useTable(
    {
      columns,
      data,
      initialState: {
        sortBy: useMemo(() => {
          return [{ id: sortHeader, desc: sortDirection === 'desc' }];
        }, [sortHeader, sortDirection]),
      },
      disableMultiSort: true,
      // manualSortBy: true,
    },
    useSortBy,
  );

  const { sortBy } = tableState;

  useEffect(() => {
    const column = sortBy[0];
    if (column !== undefined) {
      if (column.id !== sortHeader || column.desc !== (sortDirection === 'desc')) {
        onSort(column.id, column.desc);
      }
    } else {
      onSort(undefined);
    }
  }, [sortBy, sortHeader, sortDirection]);


  return (
    <div className={baseClass}>
      <div className={'data-table data-table__wrapper'}>
        {isLoading &&
          <div className={'loading-overlay'}>
            <Spinner />
          </div>
        }
        {/* TODO: can this be memoized? seems performance heavy */}
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
  columns: PropTypes.arrayOf(PropTypes.object), // TODO: create proper interface for this
  data: PropTypes.arrayOf(PropTypes.object), // TODO: create proper interface for this
  isLoading: PropTypes.bool,
  sortHeader: PropTypes.string,
  sortDirection: PropTypes.string,
  onSort: PropTypes.func,
  fetchDataAction: PropTypes.func,
};

export default DataTable;
