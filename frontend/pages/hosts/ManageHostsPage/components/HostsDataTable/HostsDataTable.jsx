import React, { useMemo, useEffect, useRef, useCallback, useState } from 'react';
import PropTypes from 'prop-types';
import { useTable, useGlobalFilter, useSortBy, useAsyncDebounce } from 'react-table';
import { useSelector, useDispatch } from 'react-redux';

// TODO: move this file closer to HostsDataTable
import { getHostTableData } from 'redux/nodes/components/ManageHostsPage/actions';

import Spinner from 'components/loaders/Spinner';
import HostPagination from 'components/hosts/HostPagination';
import scrollToTop from '../../../../../utilities/scroll_to_top';

// TODO: pass in as props
const DEFAULT_PAGE_SIZE = 100;
const DEFAULT_PAGE_INDEX = 0;
const DEBOUNCE_QUERY_DELAY = 300;
const DEFAULT_SORT_KEY = 'hostname';

// TODO: possibly get rid of this.
const containerClass = 'host-container';

const generateHostCountText = (pageIndex, itemsPerPage, resultsCount) => {
  if (itemsPerPage === resultsCount) return `${itemsPerPage}+ hosts`;

  if (pageIndex !== 0 && (resultsCount <= itemsPerPage)) return `${itemsPerPage}+ hosts`;

  return `${resultsCount} hosts`;
};

// This data table uses react-table for implementation. The relevant documentation of the library
// can be found here https://react-table.tanstack.com/docs/api/useTable
const HostsDataTable = (props) => {
  const {
    // selectedFilter is passed from parent, as it ultimately comes from the router and this
    // component cannot access the router state.
    selectedFilter,
    searchQuery,
    hiddenColumns,
    tableColumns,
  } = props;

  const [pageSize] = useState(DEFAULT_PAGE_SIZE);
  const [pageIndex, setPageIndex] = useState(DEFAULT_PAGE_INDEX);

  const dispatch = useDispatch();
  const loadingHosts = useSelector(state => state.entities.hosts.loading);
  const hosts = useSelector(state => state.entities.hosts.data);
  const hostAPIOrder = useSelector(state => state.entities.hosts.originalOrder);

  // This variable is used to keep the react-table state persistent across server calls for new data.
  // You can read more about this here technique here:
  // https://react-table.tanstack.com/docs/faq#how-do-i-stop-my-table-state-from-automatically-resetting-when-my-data-changes
  const skipPageResetRef = useRef();

  const pageIndexChangeRef = useRef();

  const columns = useMemo(() => {
    return tableColumns;
  }, [tableColumns]);

  const data = useMemo(() => {
    return hostAPIOrder.map((id) => {
      return hosts[id];
    });
  }, [hosts, hostAPIOrder]);

  const {
    headerGroups,
    rows,
    prepareRow,
    setGlobalFilter,
    setHiddenColumns,
    state: tableState,
  } = useTable(
    {
      columns,
      data,
      initialState: {
        sortBy: [{ id: DEFAULT_SORT_KEY, desc: true }],
        hiddenColumns,
      },
      autoResetHiddenColumns: false,
      disableMultiSort: true,
      manualGlobalFilter: true,
      manualSortBy: true,
      autoResetSortBy: !skipPageResetRef.current,
      autoResetGlobalFilter: !skipPageResetRef.current,
    },
    useGlobalFilter,
    useSortBy,
  );
  const { globalFilter, sortBy } = tableState;

  const debouncedGlobalFilter = useAsyncDebounce((value) => {
    skipPageResetRef.current = true;
    setGlobalFilter(value || undefined);
  }, DEBOUNCE_QUERY_DELAY);

  const onPaginationChange = useCallback((newPage) => {
    if (newPage > pageIndex) {
      setPageIndex(pageIndex + 1);
    } else {
      setPageIndex(pageIndex - 1);
    }
    pageIndexChangeRef.current = true;
    scrollToTop();
  }, [pageIndex, setPageIndex]);

  // Since searchQuery is passed in from the parent, we want to debounce the globalFilter change
  // when we see it change.
  useEffect(() => {
    debouncedGlobalFilter(searchQuery);
  }, [debouncedGlobalFilter, searchQuery]);

  // Track hidden columns changing and update the table accordingly.
  useEffect(() => {
    setHiddenColumns(hiddenColumns);
  }, [setHiddenColumns, hiddenColumns]);

  // Any changes to these relevant table search params will fire off an action to get the new
  // hosts data.
  useEffect(() => {
    if (pageIndexChangeRef.current) { // the pageIndex has changed
      dispatch(getHostTableData(pageIndex, pageSize, selectedFilter, globalFilter, sortBy));
    } else { // something besides pageIndex changed. we want to get results starting at the first page
      // NOTE: currently this causes the request to fire twice if the user is not on the first page
      // of results. Need to come back to this and figure out how to get it to
      // only fire once.
      setPageIndex(0);
      dispatch(getHostTableData(0, pageSize, selectedFilter, globalFilter, sortBy));
    }
    skipPageResetRef.current = false;
    pageIndexChangeRef.current = false;
  }, [dispatch, pageIndex, pageSize, selectedFilter, globalFilter, sortBy]);

  // No hosts for this result.
  if (!loadingHosts && Object.values(hosts).length === 0) {
    return (
      <div className={`${containerClass}  ${containerClass}--no-hosts`}>
        <div className={`${containerClass}--no-hosts__inner`}>
          <div className={'no-filter-results'}>
            <h1>No hosts match the current criteria</h1>
            <p>Expecting to see new hosts? Try again in a few seconds as the system catches up</p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className={'host-data-table'}>
      <div className={'manage-hosts__topper'}>
        <p className={'manage-hosts__host-count'}>{generateHostCountText(pageIndex, pageSize, rows.length)}</p>
      </div>
      <div className={'hosts-table hosts-table__wrapper'}>
        {loadingHosts &&
          <div className={'loading-overlay'}>
            <Spinner />
          </div>
        }
        <table className={'hosts-table__table'}>
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

      <HostPagination
        hostsOnCurrentPage={rows.length}
        currentPage={pageIndex}
        hostsPerPage={pageSize}
        onPaginationChange={onPaginationChange}
      />
    </div>
  );
};

HostsDataTable.propTypes = {
  selectedFilter: PropTypes.string,
  searchQuery: PropTypes.string,
  tableColumns: PropTypes.arrayOf(PropTypes.object), // TODO: create proper interface for this
  hiddenColumns: PropTypes.arrayOf(PropTypes.string),
};

export default HostsDataTable;
