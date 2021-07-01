import React, { useMemo, useEffect, useCallback } from "react";
import PropTypes from "prop-types";
import { useTable, useSortBy, useRowSelect } from "react-table";

import Spinner from "components/loaders/Spinner";
import Button from "../../buttons/Button";

const baseClass = "data-table-container";

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
    onSelectActionClick,
    selectActionButtonText,
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
    selectedFlatRows,
    toggleAllRowsSelected,
    state: tableState,
  } = useTable(
    {
      columns,
      data,
      initialState: {
        sortBy: useMemo(() => {
          return [{ id: sortHeader, desc: sortDirection === "desc" }];
        }, [sortHeader, sortDirection]),
      },
      disableMultiSort: true,
    },
    useSortBy,
    useRowSelect
  );

  const { sortBy, selectedRowIds } = tableState;

  // This is used to listen for changes to sort. If there is a change
  // Then the sortHandler change is fired.
  useEffect(() => {
    const column = sortBy[0];
    if (column !== undefined) {
      if (
        column.id !== sortHeader ||
        column.desc !== (sortDirection === "desc")
      ) {
        onSort(column.id, column.desc);
      }
    } else {
      onSort(undefined);
    }
  }, [sortBy, sortHeader, onSort, sortDirection]);

  const onSelectActionButtonClick = useCallback(() => {
    const entityIds = selectedFlatRows.map((row) => row.original.id);
    onSelectActionClick(entityIds);
  }, [onSelectActionClick, selectedFlatRows]);

  const onClearSelectionClick = useCallback(() => {
    toggleAllRowsSelected(false);
  }, [toggleAllRowsSelected]);

  const generateButtonText = (selectActionButtonText = "Transfer to team") => {
    return selectActionButtonText;
  };

  return (
    <div className={baseClass}>
      <div className={"data-table data-table__wrapper"}>
        {isLoading && (
          <div className={"loading-overlay"}>
            <Spinner />
          </div>
        )}
        <table className={"data-table__table"}>
          {Object.keys(selectedRowIds).length !== 0 && (
            <thead className={"active-selection"}>
              <tr {...headerGroups[0].getHeaderGroupProps()}>
                <th
                  {...headerGroups[0].headers[0].getHeaderProps(
                    headerGroups[0].headers[0].getSortByToggleProps()
                  )}
                >
                  {headerGroups[0].headers[0].render("Header")}
                </th>
                <th className={"active-selection__container"}>
                  <div className={"active-selection__inner"}>
                    <p>
                      <span>{selectedFlatRows.length}</span> selected
                    </p>
                    <Button
                      onClick={onClearSelectionClick}
                      variant={"text-link"}
                    >
                      Clear selection
                    </Button>
                    <Button onClick={onSelectActionButtonClick}>
                      {generateButtonText(selectActionButtonText)}
                    </Button>
                  </div>
                </th>
              </tr>
            </thead>
          )}
          <thead>
            {headerGroups.map((headerGroup) => (
              <tr {...headerGroup.getHeaderGroupProps()}>
                {headerGroup.headers.map((column) => (
                  <th {...column.getHeaderProps(column.getSortByToggleProps())}>
                    {column.render("Header")}
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
                      <td {...cell.getCellProps()}>{cell.render("Cell")}</td>
                    );
                  })}
                </tr>
              );
            })}
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
  onSelectActionClick: PropTypes.func,
};

export default DataTable;
