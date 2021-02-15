import React, { useState, useCallback, useEffect } from 'react';
import PropTypes from 'prop-types';
import { useTable, useGlobalFilter, useSortBy, useAsyncDebounce } from 'react-table';

// TODO: move this file closer to HostsDataTable
import hostInterface from 'interfaces/host';
import { humanHostMemory, humanHostUptime } from 'kolide/helpers';
import InputField from 'components/forms/fields/InputField';

import TextCell from '../TextCell/TextCell';
import StatusCell from '../StatusCell/StatusCell';
import LinkCell from '../LinkCell/LinkCell';

const useListenForTableChanges = (tableState, changeTableDataHandler) => {
  useEffect(() => {
    // changeTableDataHandler();
    console.log('TABLE STATE:', tableState);
  }, [tableState.sortBy, tableState.globalFilter]);
};

// This data table uses react-table for implementation. The relevant documentation of the library
// can be found here https://react-table.tanstack.com/docs/api/useTable
const HostsDataTable = (props) => {
  console.log('render HostsDataTable');
  const columns = React.useMemo(() => {
    return [
      { Header: 'Hostname', accessor: 'hostname', Cell: cellProps => <LinkCell value={cellProps.cell.value} host={cellProps.row.original} /> },
      { Header: 'Status', accessor: 'status', Cell: cellProps => <StatusCell value={cellProps.cell.value} /> },
      { Header: 'OS', accessor: 'os_version', Cell: cellProps => <TextCell value={cellProps.cell.value} /> },
      { Header: 'Osquery', accessor: 'osquery_version', Cell: cellProps => <TextCell value={cellProps.cell.value} /> },
      { Header: 'IPv4', accessor: 'primary_ip', Cell: cellProps => <TextCell value={cellProps.cell.value} /> },
      { Header: 'Physical Address', accessor: 'primary_mac', Cell: cellProps => <TextCell value={cellProps.cell.value} /> },
      { Header: 'CPU', accessor: 'host_cpu', Cell: cellProps => <TextCell value={cellProps.cell.value} /> },
      { Header: 'Memory', accessor: 'memory', Cell: cellProps => <TextCell value={cellProps.cell.value} formatter={humanHostMemory} /> },
      { Header: 'Uptime', accessor: 'uptime', Cell: cellProps => <TextCell value={cellProps.cell.value} formatter={humanHostUptime} /> },
    ];
  }, []);

  const { hosts } = props;
  const data = React.useMemo(() => {
    return hosts;
  }, [hosts]);

  const {
    headerGroups,
    rows,
    prepareRow,
    setGlobalFilter,
    state,
  } = useTable({ columns, data }, useGlobalFilter, useSortBy);

  const [searchQuery, setSearchQuery] = useState('');

  const debouncedGlobalFilter = useAsyncDebounce((value) => {
    setGlobalFilter(value || undefined);
  }, 200);

  const onChange = useCallback((value) => {
    setSearchQuery(value);
    debouncedGlobalFilter(value);
  }, [setSearchQuery, debouncedGlobalFilter]);

  const { onChangeTableData } = props;
  useListenForTableChanges(state, onChangeTableData);

  return (
    <React.Fragment>
      <InputField
        placeholder="Search hosts by hostname"
        name=""
        onChange={onChange}
        value={searchQuery}
        inputWrapperClass={'host-side-panel__filter-labels'}
      />

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
            })}
          </tbody>
        </table>
      </div>
    </React.Fragment>
  );
};

HostsDataTable.propTypes = {
  hosts: PropTypes.arrayOf(hostInterface),
};

export default HostsDataTable;
