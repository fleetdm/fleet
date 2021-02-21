import React, { useMemo, useEffect, useRef, useCallback, useState } from 'react';
import PropTypes from 'prop-types';
import {useTable, useGlobalFilter, useSortBy, useAsyncDebounce } from 'react-table';
import { useSelector, useDispatch } from 'react-redux';


// TODO: move this file closer to HostsDataTable
import { humanHostMemory, humanHostUptime } from 'kolide/helpers';
import { getHostTableData } from 'redux/nodes/components/ManageHostsPage/actions';

import Spinner from 'components/loaders/Spinner';
import HostPagination from 'components/hosts/HostPagination';

import HeaderCell from '../HeaderCell/HeaderCell';
import TextCell from '../TextCell/TextCell';
import StatusCell from '../StatusCell/StatusCell';
import LinkCell from '../LinkCell/LinkCell';

// TODO: pass in as props
const DEFAULT_PAGE_SIZE = 2;
const DEFAULT_PAGE_INDEX = 0;
const DEBOUNCE_QUERY_DELAY = 300;
const DEFAULT_SORT_KEY = 'hostname';
const DEFAULT_SORT_DIRECTION = 'ASC';

// TODO: possibly get rid of this.
const containerClass = 'host-container';

// This data table uses react-table for implementation. The relevant documentation of the library
// can be found here https://react-table.tanstack.com/docs/api/useTable
const HostsDataTable = (props) => {
  const {
    // selectedFilter is passed from parent, as it ultimately comes from the router and this
    // component cannot access the router state.
    selectedFilter,
    searchQuery,
  } = props;

  const [pageSize] = useState(DEFAULT_PAGE_SIZE);
  const [pageIndex, setPageIndex] = useState(DEFAULT_PAGE_INDEX);

  const dispatch = useDispatch();
  const loadingHosts = useSelector(state => state.entities.hosts.loading);
  const hosts = useSelector(state => state.entities.hosts.data);

  // This variable is used to keep the react-table state persistant across server calls for new data.
  // You can read more about this here technique here:
  // https://react-table.tanstack.com/docs/faq#how-do-i-stop-my-table-state-from-automatically-resetting-when-my-data-changes
  const skipPageResetRef = useRef();

  const pageIndexChangeRef = useRef();

  // TODO: maybe pass as props?
  const columns = useMemo(() => {
    return [
      { Header: cellProps => <HeaderCell all={cellProps.column} value={'Hostname'} isSortedDesc={cellProps.column.isSortedDesc} />, accessor: 'hostname', Cell: cellProps => <LinkCell value={cellProps.cell.value} host={cellProps.row.original} /> },
      { Header: 'Status', disableSortBy: true, accessor: 'status', Cell: cellProps => <StatusCell value={cellProps.cell.value} /> },
      { Header: cellProps => <HeaderCell all={cellProps.column} value={'OS'} isSortedDesc={cellProps.column.isSortedDesc} />, accessor: 'os_version', Cell: cellProps => <TextCell value={cellProps.cell.value} /> },
      { Header: cellProps => <HeaderCell value={'Osquery'} isSortedDesc={cellProps.column.isSortedDesc} />, accessor: 'osquery_version', Cell: cellProps => <TextCell value={cellProps.cell.value} /> },
      { Header: cellProps => <HeaderCell value={'IPv4'} isSortedDesc={cellProps.column.isSortedDesc} />, accessor: 'primary_ip', Cell: cellProps => <TextCell value={cellProps.cell.value} /> },
      { Header: cellProps => <HeaderCell value={'Physical Address'} isSortedDesc={cellProps.column.isSortedDesc} />, accessor: 'primary_mac', Cell: cellProps => <TextCell value={cellProps.cell.value} /> },
      { Header: 'CPU', disableSortBy: true, accessor: 'host_cpu', Cell: cellProps => <TextCell value={cellProps.cell.value} /> },
      { Header: cellProps => <HeaderCell value={'Memory'} isSortedDesc={cellProps.column.isSortedDesc} />, accessor: 'memory', Cell: cellProps => <TextCell value={cellProps.cell.value} formatter={humanHostMemory} /> },
      { Header: cellProps => <HeaderCell value={'Uptime'} isSortedDesc={cellProps.column.isSortedDesc} />, accessor: 'uptime', Cell: cellProps => <TextCell value={cellProps.cell.value} formatter={humanHostUptime} /> },
    ];
  }, []);

  const data = useMemo(() => {
    return Object.values(hosts);
  }, [hosts]);

  const {
    headerGroups,
    rows,
    prepareRow,
    setGlobalFilter,
    state: tableState,
  } = useTable(
    { columns,
      data,
      initialState: {
        sortBy: [{ id: DEFAULT_SORT_KEY, desc: DEFAULT_SORT_DIRECTION === 'DESC' }],
      },
      disableMultiSort: true,
      manualGlobalFilter: true,
      autoResetSortBy: skipPageResetRef.current,
      autoResetGlobalFilter: skipPageResetRef.current,
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
      // pageIndexChangeRef.current = pageIndex;
      setPageIndex(pageIndex + 1);
    } else {
      // pageIndpageIndexChangeRefexRef.current = pageIndex;
      setPageIndex(pageIndex - 1);
    }
    pageIndexChangeRef.current = true;
    // scrollToTop();
  }, [pageIndex, setPageIndex]);

  // Since searchQuery is feed in from the parent, we want to debounce the globalfilter change
  // when we see it change.
  useEffect(() => {
    debouncedGlobalFilter(searchQuery);
  }, [debouncedGlobalFilter, searchQuery]);

  // Any changes to these relevent table search params will fire off an action to get the new
  // hosts data.
  useEffect(() => {
    if (pageIndexChangeRef.current) { // the pageIndex has changed
      dispatch(getHostTableData(pageIndex, pageSize, selectedFilter, globalFilter, sortBy));
    } else {
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
          <div>
            <h1>No hosts match the current search criteria</h1>
            <p>Expecting to see new hosts? Try again in a few seconds as the system catches up</p>
          </div>
        </div>

        <HostPagination
          hostOnCurrentPage={100}
          currentPage={pageIndex}
          hostsPerPage={pageSize}
          onPaginationChange={onPaginationChange}
        />
      </div>
    );
  }

  console.log(rows);
  return (
    <React.Fragment>
      <div className={'hosts-table hosts-table__wrapper'}>
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
            {loadingHosts
              ? <tr><td><Spinner /></td></tr>
              : rows.map((row) => {
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
    </React.Fragment>
  );
};

HostsDataTable.propTypes = {
  selectedFilter: PropTypes.string,
  searchQuery: PropTypes.string,
};

export default HostsDataTable;
