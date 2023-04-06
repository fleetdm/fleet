/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React, { useMemo, useEffect, useCallback, useContext } from "react";
import { TableContext } from "context/table";
import classnames from "classnames";
import {
  Column,
  HeaderGroup,
  Row,
  useFilters,
  useGlobalFilter,
  usePagination,
  useRowSelect,
  useSortBy,
  useTable,
} from "react-table";
import { kebabCase, noop, omit, pick } from "lodash";
import { useDebouncedCallback } from "use-debounce";

import useDeepEffect from "hooks/useDeepEffect";
import sort from "utilities/sort";
import { AppContext } from "context/app";

import Button from "components/buttons/Button";
// @ts-ignore
import FleetIcon from "components/icons/FleetIcon";
import Spinner from "components/Spinner";
import { ButtonVariant } from "components/buttons/Button/Button";
import ActionButton, { IActionButtonProps } from "./ActionButton";

const baseClass = "data-table-block";

interface IDataTableProps {
  columns: Column[];
  data: any;
  filters?: Record<string, string | number | boolean>;
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
  isClientSidePagination?: boolean;
  isClientSideFilter?: boolean;
  disableHighlightOnHover?: boolean;
  searchQuery?: string;
  searchQueryColumn?: string;
  selectedDropdownFilter?: string;
  onSelectSingleRow?: (value: Row) => void;
  onResultsCountChange?: (value: number) => void;
  renderFooter?: () => JSX.Element | null;
  renderPagination?: () => JSX.Element | null;
  setExportRows?: (rows: Row[]) => void;
}

interface IHeaderGroup extends HeaderGroup {
  title?: string;
}

const CLIENT_SIDE_DEFAULT_PAGE_SIZE = 20;

// This data table uses react-table for implementation. The relevant documentation of the library
// can be found here https://react-table.tanstack.com/docs/api/useTable
const DataTable = ({
  columns: tableColumns,
  data: tableData,
  filters: tableFilters,
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
  isClientSidePagination,
  isClientSideFilter,
  disableHighlightOnHover,
  searchQuery,
  searchQueryColumn,
  selectedDropdownFilter,
  onSelectSingleRow,
  onResultsCountChange,
  renderFooter,
  renderPagination,
  setExportRows,
}: IDataTableProps): JSX.Element => {
  const { resetSelectedRows } = useContext(TableContext);
  const { isOnlyObserver } = useContext(AppContext);

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
    // pageOptions,
    // pageCount,
    // gotoPage,
    nextPage,
    previousPage,
    setPageSize,
    setFilter, // sets a specific column-level filter
    setAllFilters, // sets all of the column-level filters; rows are included in filtered results only if each column filter return true
    setGlobalFilter, // sets the global filter; this serves as a global free text search across all columns (excluding only those where `disableGlobalFilter: true`)
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
      // Expands the enumerated `filterTypes` for react-table
      // (see https://github.com/TanStack/react-table/blob/alpha/packages/react-table/src/filterTypes.ts)
      // with custom `filterTypes` defined for this `useTable` instance
      filterTypes: React.useMemo(
        () => ({
          hasLength: (
            // eslint-disable-next-line @typescript-eslint/no-shadow
            rows: Row[],
            columnIds: string[],
            filterValue: boolean
          ) => {
            return !filterValue
              ? rows
              : rows?.filter((row) => {
                  return columnIds?.some((id) => row?.values?.[id]?.length);
                });
          },
        }),
        []
      ),
      autoResetFilters: false,
      // Expands the enumerated `sortTypes` for react-table
      // (see https://github.com/tannerlinsley/react-table/blob/master/src/sortTypes.js)
      // with custom `sortTypes` defined for this `useTable` instance
      sortTypes: React.useMemo(
        () => ({
          boolean: (
            a: { values: Record<string, unknown> },
            b: { values: Record<string, unknown> },
            id: string
          ) => sort.booleanAsc(a.values[id], b.values[id]),

          caseInsensitive: (
            a: { values: Record<string, unknown> },
            b: { values: Record<string, unknown> },
            id: string
          ) => sort.caseInsensitiveAsc(a.values[id], b.values[id]),

          dateStrings: (
            a: { values: Record<string, string> },
            b: { values: Record<string, string> },
            id: string
          ) => sort.dateStringsAsc(a.values[id], b.values[id]),

          hasLength: (
            a: { values: Record<string, unknown[]> },
            b: { values: Record<string, unknown[]> },
            id: string
          ) => {
            return sort.hasLength(a.values[id], b.values[id]);
          },
        }),
        []
      ),
    },
    useGlobalFilter, // order of these hooks matters; here we first apply the global filter (if any); this could be reversed depending on where we want to target performance
    useFilters, // react-table applies column-level filters after first applying the global filter (if any)
    useSortBy,
    usePagination,
    useRowSelect
  );

  const { sortBy, selectedRowIds } = tableState;

  useEffect(() => {
    if (tableFilters) {
      const filtersToSet = tableFilters;
      const global = filtersToSet.global;
      setGlobalFilter(global);
      delete filtersToSet.global;
      const allFilters = Object.entries(filtersToSet).map(([id, value]) => ({
        id,
        value,
      }));
      !!allFilters.length && setAllFilters(allFilters);
      setExportRows && setExportRows(rows);
    }
  }, [tableFilters]);

  useEffect(() => {
    setExportRows && setExportRows(rows);
  }, [tableState.filters, rows.length]);

  // Listen for changes to filters if clientSideFilter is enabled

  const setDebouncedClientFilter = useDebouncedCallback(
    (column: string, query: string) => {
      setFilter(column, query);
    },
    300
  );

  useEffect(() => {
    if (isClientSideFilter && onResultsCountChange) {
      onResultsCountChange(rows.length);
    }
  }, [isClientSideFilter, onResultsCountChange, rows.length]);

  useEffect(() => {
    if (isClientSideFilter && searchQueryColumn) {
      setDebouncedClientFilter(searchQueryColumn, searchQuery || "");
    }
  }, [searchQuery, searchQueryColumn]);

  useEffect(() => {
    if (isClientSideFilter && selectedDropdownFilter) {
      selectedDropdownFilter === "all"
        ? setDebouncedClientFilter("platforms", "")
        : setDebouncedClientFilter("platforms", selectedDropdownFilter);
    }
  }, [selectedDropdownFilter]);

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
  }, [isAllPagesSelected, toggleAllRowsSelected]);

  useEffect(() => {
    setPageSize(defaultPageSize || CLIENT_SIDE_DEFAULT_PAGE_SIZE);
  }, [setPageSize]);

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
  }, [toggleAllPagesSelected, toggleAllRowsSelected]);

  const onSingleRowClick = useCallback(
    (row) => {
      if (disableMultiRowSelect) {
        row.toggleRowSelected();
        onSelectSingleRow && onSelectSingleRow(row);
        toggleAllRowsSelected(false);
      }
    },
    [disableMultiRowSelect, onSelectSingleRow, toggleAllRowsSelected]
  );

  const renderColumnHeader = (column: IHeaderGroup) => {
    // if there is a column filter, we want the `onClick` event listener attached
    // just to the child title span so that clicking into the column filter input
    // doesn't also sort the column
    const spanProps = column.Filter
      ? pick(column.getSortByToggleProps(), "onClick")
      : {};

    return (
      <div className="column-header">
        <span {...spanProps}>{column.render("Header")}</span>
        {column.Filter && column.render("Filter")}
      </div>
    );
  };

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

  const pageOrRows = isClientSidePagination ? page : rows;

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

  const tableStyles = classnames({
    "data-table__table": true,
    "is-observer": isOnlyObserver,
  });

  return (
    <div className={baseClass}>
      {isLoading && (
        <div className={"loading-overlay"}>
          <Spinner />
        </div>
      )}
      <div className={"data-table data-table__wrapper"}>
        <table className={tableStyles}>
          {Object.keys(selectedRowIds).length !== 0 && (
            <thead className={"active-selection"}>
              <tr {...headerGroups[0].getHeaderGroupProps()}>
                <th
                  className={"active-selection__checkbox"}
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
                {headerGroup.headers.map((column) => {
                  let thProps = column.getSortByToggleProps({
                    title: undefined,
                  });
                  if (column.Filter) {
                    // if there is a column filter, we want the `onClick` event listener attached
                    // just to the child title span so that clicking into the column filter input
                    // doesn't also sort the column
                    thProps = omit(thProps, "onClick");
                  }

                  return (
                    <th
                      key={column.id}
                      className={column.id ? `${column.id}__header` : ""}
                      {...thProps}
                    >
                      {renderColumnHeader(column)}
                    </th>
                  );
                })}
              </tr>
            ))}
          </thead>
          <tbody>
            {pageOrRows.map((row: Row) => {
              prepareRow(row);

              const rowStyles = classnames({
                "single-row": disableMultiRowSelect,
                "disable-highlight": disableHighlightOnHover,
              });
              return (
                <tr
                  className={rowStyles}
                  {...row.getRowProps({
                    // @ts-ignore // TS complains about prop not existing
                    onClick: () => {
                      onSingleRowClick &&
                        disableMultiRowSelect &&
                        onSingleRowClick(row);
                    },
                  })}
                >
                  {row.cells.map((cell: any) => {
                    return (
                      <td
                        key={cell.column.id}
                        className={
                          cell.column.id ? `${cell.column.id}__cell` : ""
                        }
                        {...cell.getCellProps()}
                      >
                        {cell.render("Cell")}
                      </td>
                    );
                  })}
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
      <div className={`${baseClass}__footer`}>
        {renderFooter && (
          <div className={`${baseClass}__footer-text`}>{renderFooter()}</div>
        )}
        {isClientSidePagination ? (
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
        ) : (
          renderPagination && renderPagination()
        )}
      </div>
    </div>
  );
};

export default DataTable;
