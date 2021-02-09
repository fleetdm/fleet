import React from 'react';
import { useTable } from 'react-table';

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
    getTableProps,
    getTableBodyProps,
    headerGroups,
    rows,
    prepareRow,
  } = useTable({ columns, data });

  return (
    <div className={`${baseClass} ${baseClass}__wrapper`}>
      <table {...getTableProps()} className={`${baseClass}__table`}>
        <thead>
          {headerGroups.map(headerGroup => (
            <tr {...headerGroup.getHeaderGroupProps()}>
              {headerGroup.headers.map(column => (
                <th>
                  {column.render('Header')}
                </th>
              ))}
            </tr>
          ))}
        </thead>
        <tbody {...getTableBodyProps()}>
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
