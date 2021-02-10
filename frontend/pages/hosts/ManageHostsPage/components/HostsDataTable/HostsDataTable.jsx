import React from 'react';
import { useTable } from 'react-table';

import TableHeader from '../TableHeader/TableHeader';
import TextCell from '../TextCell/TextCell';

const baseClass = 'hosts-table';

const HostsDataTable = ({ hosts }) => {
  const columns = React.useMemo(() => {
    return [
      { Header: 'Hostname', accessor: 'hostname' },
      { Header: 'Status', accessor: 'status' },
      { Header: 'OS', accessor: 'os_version' },
      { Header: 'Osquery', accessor: 'osquery_version' },
      { Header: 'IPv4', accessor: 'primary_ip' },
      { Header: 'Physical Address', accessor: 'primary_mac' },
      { Header: 'CPU', accessor: 'host_cpu' },
      { Header: 'Memory', accessor: 'memory' },
      { Header: 'Uptime', accessor: 'uptime' },
    ];
  }, []);

  const data = React.useMemo(() => {
    return hosts;
  }, [hosts]);


  const {
    headerGroups,
    rows,
    prepareRow,
  } = useTable({ columns, data, defaultColumn: { Cell: TextCell } });

  return (
    <div className={`${baseClass} ${baseClass}__wrapper`}>
      <table className={`${baseClass}__table`}>
        <thead>
          {headerGroups.map(headerGroup => (
            <tr {...headerGroup.getHeaderGroupProps()}>
              {headerGroup.headers.map(column => (
                <th {...column.getHeaderProps()}>{column.render('Header')}</th>
                // <TableHeader title={column.render('Header')} />
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
  );
};

export default HostsDataTable;
