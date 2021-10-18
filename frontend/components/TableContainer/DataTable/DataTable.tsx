import React, { useMemo, useEffect, useCallback, useContext } from "react";
import { TableContext } from "context/table";
import PropTypes from "prop-types";
import classnames from "classnames";
import {
  useTable,
  useSortBy,
  useRowSelect,
  Row,
  usePagination,
} from "react-table";
import { isString, kebabCase, noop } from "lodash";

import { useDeepEffect } from "utilities/hooks";
import sort from "utilities/sort";

import Button from "components/buttons/Button";
// @ts-ignore
import FleetIcon from "components/icons/FleetIcon";
import Spinner from "components/loaders/Spinner";
import { ButtonVariant } from "components/buttons/Button/Button";
import ActionButton, { IActionButtonProps } from "./ActionButton";

const baseClass = "data-table-container";

interface IDataTableProps {
  columns: any;
  data: any;
  isLoading: boolean;
  manualSortBy?: boolean;
  sortHeader: any;
  sortDirection: any;
  onSort: any; // TODO: an event type
  disableMultiRowSelect: boolean;
  showMarkAllPages: boolean;
  isAllPagesSelected: boolean; // TODO: make dependent on showMarkAllPages
  toggleAllPagesSelected?: any; // TODO: an event type and make it dependent on showMarkAllPages
  resultsTitle: string;
  defaultPageSize: number;
  primarySelectActionButtonVariant?: ButtonVariant;
  primarySelectActionButtonIcon?: string;
  primarySelectActionButtonText?: string | ((targetIds: number[]) => string);
  onPrimarySelectActionClick: any; // figure out type
  secondarySelectActions?: IActionButtonProps[];
  onSelectSingleRow?: (value: Row) => void;
  clientSidePagination?: boolean;
}

const CLIENT_SIDE_DEFAULT_PAGE_SIZE = 20;

// This data table uses react-table for implementation. The relevant documentation of the library
// can be found here https://react-table.tanstack.com/docs/api/useTable
const DataTable = ({
  columns: tableColumns,
  data: tableData,
  isLoading,
  manualSortBy = false,
  sortHeader,
  sortDirection,
  onSort,
  disableMultiRowSelect,
  showMarkAllPages,
  isAllPagesSelected,
  toggleAllPagesSelected,
  resultsTitle,
  defaultPageSize,
  primarySelectActionButtonIcon,
  primarySelectActionButtonVariant,
  onPrimarySelectActionClick,
  primarySelectActionButtonText,
  secondarySelectActions,
  onSelectSingleRow,
  clientSidePagination,
}: IDataTableProps): JSX.Element => {
  const { resetSelectedRows } = useContext(TableContext);

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
    page, // Instead of using 'rows', we'll use page,
    // which has only the rows for the active page

    // The rest of these things are super handy, too ;)
    canPreviousPage,
    canNextPage,
    pageOptions,
    pageCount,
    gotoPage,
    nextPage,
    previousPage,
    setPageSize,
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
      disableSortRemove: true,
      manualSortBy,
      // Initializes as false, but changes briefly to true on successful notification
      autoResetSelectedRows: resetSelectedRows,
      sortTypes: React.useMemo(
        () => ({
          caseInsensitive: (a: any, b: any, id: any) => {
            let valueA = a.values[id];
            let valueB = b.values[id];

            valueA = isString(valueA) ? valueA.toLowerCase() : valueA;
            valueB = isString(valueB) ? valueB.toLowerCase() : valueB;

            if (valueB > valueA) {
              return -1;
            }
            if (valueB < valueA) {
              return 1;
            }
            return 0;
          },
          dateStrings: (a: any, b: any, id: any) =>
            sort.dateStringsAsc(a.values[id], b.values[id]),
        }),
        []
      ),
    },
    useSortBy,
    usePagination,
    useRowSelect
  );

  const { sortBy, selectedRowIds, pageIndex, pageSize } = tableState;

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

  const onToggleAllPagesClick = useCallback(() => {
    toggleAllPagesSelected();
  }, [toggleAllPagesSelected]);

  const onClearSelectionClick = useCallback(() => {
    toggleAllRowsSelected(false);
    toggleAllPagesSelected(false);
  }, [toggleAllRowsSelected]);

  const onSingleRowClick = useCallback(
    (row) => {
      if (disableMultiRowSelect) {
        row.toggleRowSelected();
        onSelectSingleRow && onSelectSingleRow(row);
        toggleAllRowsSelected(false);
      }
    },
    [disableMultiRowSelect]
  );

  const renderSelectedCount = (): JSX.Element => {
    return (
      <p>
        <span>
          {selectedFlatRows.length}
          {isAllPagesSelected && "+"}
        </span>{" "}
        selected
      </p>
    );
  };

  const renderAreAllSelected = (): JSX.Element | null => {
    if (isAllPagesSelected) {
      return <p>All matching {resultsTitle} are selected</p>;
    }

    if (isAllRowsSelected) {
      return <p>All {resultsTitle} on this page are selected</p>;
    }
    return null;
  };

  const renderActionButton = (
    actionButtonProps: IActionButtonProps
  ): JSX.Element => {
    const {
      name,
      onActionButtonClick,
      buttonText,
      targetIds,
      variant,
      hideButton,
      icon,
      iconPosition,
    } = actionButtonProps;
    return (
      <div className={`${baseClass}__${kebabCase(name)}`}>
        <ActionButton
          key={kebabCase(name)}
          name={name}
          buttonText={buttonText}
          onActionButtonClick={onActionButtonClick || noop}
          targetIds={targetIds}
          variant={variant}
          hideButton={hideButton}
          icon={icon}
          iconPosition={iconPosition}
        />
      </div>
    );
  };

  const renderPrimarySelectAction = (): JSX.Element | null => {
    const targetIds = selectedFlatRows.map((row: any) => row.original.id);
    const buttonText =
      typeof primarySelectActionButtonText === "function"
        ? primarySelectActionButtonText(targetIds)
        : primarySelectActionButtonText;
    const name = buttonText ? kebabCase(buttonText) : "primary-select-action";

    const actionProps = {
      name,
      buttonText: buttonText || "",
      onActionButtonClick: onPrimarySelectActionClick,
      targetIds,
      variant: primarySelectActionButtonVariant,
      icon: primarySelectActionButtonIcon,
    };

    return !buttonText ? null : renderActionButton(actionProps);
  };

  const renderSecondarySelectActions = (): JSX.Element[] | null => {
    if (secondarySelectActions) {
      const targetIds = selectedFlatRows.map((row: any) => row.original.id);
      const buttons = secondarySelectActions.map((actionProps) => {
        actionProps = { ...actionProps, targetIds };
        return renderActionButton(actionProps);
      });
      return buttons;
    }
    return null;
  };

  const shouldRenderToggleAllPages =
    Object.keys(selectedRowIds).length >= defaultPageSize &&
    showMarkAllPages &&
    !isAllPagesSelected;

  const pageOrRows = clientSidePagination ? page : rows;

  useEffect(() => {
    setPageSize(CLIENT_SIDE_DEFAULT_PAGE_SIZE);
  }, []);

  const previousButton = (
    <>
      <FleetIcon name="chevronleft" /> Previous
    </>
  );
  const nextButton = (
    <>
      Next <FleetIcon name="chevronright" />
    </>
  );
  return (
    <div className={baseClass}>
      {isLoading && (
        <div className={"loading-overlay"}>
          <Spinner />
        </div>
      )}
      <div className={"data-table data-table__wrapper"}>
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
                    {renderSelectedCount()}
                    <div className={"active-selection__inner-left"}>
                      {secondarySelectActions && renderSecondarySelectActions()}
                    </div>
                    <div className={"active-selection__inner-right"}>
                      {primarySelectActionButtonText &&
                        renderPrimarySelectAction()}
                    </div>
                    {toggleAllPagesSelected && renderAreAllSelected()}
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
                </th>
              </tr>
            </thead>
          )}
          <thead>
            {headerGroups.map((headerGroup) => (
              <tr {...headerGroup.getHeaderGroupProps()}>
                {headerGroup.headers.map((column) => (
                  <th
                    className={column.id ? `${column.id}__header` : ""}
                    {...column.getHeaderProps(column.getSortByToggleProps())}
                  >
                    {column.render("Header")}
                  </th>
                ))}
              </tr>
            ))}
          </thead>
          <tbody>
            {pageOrRows.map((row: any) => {
              prepareRow(row);

              const rowStyles = classnames({
                "single-row": disableMultiRowSelect,
              });
              return (
                <tr
                  className={rowStyles}
                  {...row.getRowProps({
                    // @ts-ignore // TS complains about prop not existing
                    onClick: () => {
                      disableMultiRowSelect && onSingleRowClick(row);
                    },
                  })}
                >
                  {row.cells.map((cell: any) => {
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
      {clientSidePagination && (
        <div className={`${baseClass}__pagination`}>
          <Button
            variant="unstyled"
            onClick={() => previousPage()}
            disabled={!canPreviousPage}
          >
            {previousButton}
          </Button>
          <Button
            variant="unstyled"
            onClick={() => nextPage()}
            disabled={!canNextPage}
          >
            {nextButton}
          </Button>
        </div>
      )}
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
  onPrimarySelectActionClick: PropTypes.func,
  secondarySelectActions: PropTypes.arrayOf(PropTypes.object),
};

export default DataTable;
