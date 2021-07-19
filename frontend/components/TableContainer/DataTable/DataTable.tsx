import React, { useMemo, useEffect, useCallback } from "react";
import PropTypes from "prop-types";
import { useTable, useSortBy, useRowSelect } from "react-table";
import useDeepEffect from "utilities/hooks/useDeepEffect";

import Spinner from "components/loaders/Spinner";
import Button from "../../buttons/Button";

const baseClass = "data-table-container";

interface IDataTableProps {
  columns: any;
  data: any;
  isLoading: boolean;
  sortHeader: any;
  sortDirection: any;
  onSort: any; // TODO: an event type
  onSelectActionClick: any; // TODO: an event type
  showMarkAllPages: boolean;
  isAllPagesSelected: boolean; // TODO: make dependent on showMarkAllPages
  toggleAllPagesSelected?: any; // TODO: an event type and make it dependent on showMarkAllPages
  resultsTitle: string;
  defaultPageSize: number;
  selectActionButtonText?: string;
}

// This data table uses react-table for implementation. The relevant documentation of the library
// can be found here https://react-table.tanstack.com/docs/api/useTable
<<<<<<< HEAD:frontend/components/TableContainer/DataTable/DataTable.jsx
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

=======
const DataTable = ({
  columns: tableColumns,
  data: tableData,
  isLoading,
  sortHeader,
  sortDirection,
  onSort,
  onSelectActionClick,
  showMarkAllPages,
  isAllPagesSelected,
  toggleAllPagesSelected,
  resultsTitle,
  defaultPageSize,
  selectActionButtonText,
}: IDataTableProps) => {
>>>>>>> main:frontend/components/TableContainer/DataTable/DataTable.tsx
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
    isAllRowsSelected,
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

  useEffect(() => {
    if (isAllPagesSelected) {
      toggleAllRowsSelected(true);
    }
  }, [isAllPagesSelected]);

  useDeepEffect(() => {
    if (
      Object.keys(selectedRowIds).length < rows.length &&
      toggleAllPagesSelected
    ) {
      toggleAllPagesSelected(false);
    }
  }, [tableState.selectedRowIds, toggleAllPagesSelected]);

  const onSelectActionButtonClick = useCallback(() => {
    const entityIds = selectedFlatRows.map((row: any) => row.original.id);
    onSelectActionClick(entityIds);
  }, [onSelectActionClick, selectedFlatRows]);

  const onToggleAllPagesClick = useCallback(() => {
    toggleAllPagesSelected();
  }, [toggleAllPagesSelected]);

  const onClearSelectionClick = useCallback(() => {
    toggleAllRowsSelected(false);
    toggleAllPagesSelected(false);
  }, [toggleAllRowsSelected]);

<<<<<<< HEAD:frontend/components/TableContainer/DataTable/DataTable.jsx
  const generateButtonText = (selectActionButtonText = "Transfer to team") => {
    return selectActionButtonText;
  };

=======
  const renderSelectedText = (): JSX.Element => {
    if (isAllPagesSelected) {
      return <p>All matching {resultsTitle} are selected</p>;
    }

    if (isAllRowsSelected) {
      return <p>All {resultsTitle} on this page are selected</p>;
    }

    return (
      <p>
        <span>{selectedFlatRows.length}</span> selected
      </p>
    );
  };

  const generateButtonText = (text = "Transfer to team") => {
    return text;
  };

  const shouldRenderToggleAllPages =
    Object.keys(selectedRowIds).length >= defaultPageSize &&
    showMarkAllPages &&
    !isAllPagesSelected;
>>>>>>> main:frontend/components/TableContainer/DataTable/DataTable.tsx
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
<<<<<<< HEAD:frontend/components/TableContainer/DataTable/DataTable.jsx
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
=======
                    <div className={"active-selection__inner-left"}>
                      {renderSelectedText()}
                      {shouldRenderToggleAllPages && (
                        <Button
                          onClick={onToggleAllPagesClick}
                          variant={"text-link"}
                          className={"light-text"}
                        >
                          <>Select all matching {resultsTitle}</>
                        </Button>
                      )}
                      <Button
                        onClick={onClearSelectionClick}
                        variant={"text-link"}
                      >
                        Clear selection
                      </Button>
                    </div>
                    <div className={"active-selection__inner-right"}>
                      <Button onClick={onSelectActionButtonClick}>
                        {generateButtonText(selectActionButtonText)}
                      </Button>
                    </div>
>>>>>>> main:frontend/components/TableContainer/DataTable/DataTable.tsx
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
