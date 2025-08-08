/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React, {
  useMemo,
  useEffect,
  useCallback,
  useContext,
  useRef,
} from "react";
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
import { kebabCase, noop } from "lodash";
import { useDebouncedCallback } from "use-debounce";

import useDeepEffect from "hooks/useDeepEffect";
import sort from "utilities/sort";
import { AppContext } from "context/app";

import Button from "components/buttons/Button";
import Spinner from "components/Spinner";
import Pagination from "components/Pagination";
import ActionButton from "./ActionButton";
import { IActionButtonProps } from "./ActionButton/ActionButton";

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
  keyboardSelectableRows?: boolean;
  showMarkAllPages: boolean;
  isAllPagesSelected: boolean; // TODO: make dependent on showMarkAllPages
  toggleAllPagesSelected?: any; // TODO: an event type and make it dependent on showMarkAllPages
  resultsTitle?: string;
  defaultPageSize: number;
  defaultPageIndex?: number;
  defaultSelectedRows?: Record<string, boolean>;
  /** Default: true (same as useTable default)
   *  False prevents unnecessary page resets when a column ordering changes
   *  e.g. when clicking on an action that modifies the data
   */
  autoResetPage?: boolean;
  primarySelectAction?: IActionButtonProps;
  secondarySelectActions?: IActionButtonProps[];
  isClientSidePagination?: boolean;
  onClientSidePaginationChange?: (pageIndex: number) => void; // Used to set URL to correct path and include page query param
  isClientSideFilter?: boolean;
  disableHighlightOnHover?: boolean;
  searchQuery?: string;
  searchQueryColumn?: string;
  selectedDropdownFilter?: string;
  /** Set to true to persist the row selections across table data filters */
  persistSelectedRows?: boolean;
  /** Set to `true` to not display the footer section of the table */
  hideFooter?: boolean;
  onSelectSingleRow?: (value: Row) => void;
  onClickRow?: (value: any) => void;
  onResultsCountChange?: (value: number) => void;
  /** Optional help text to render on bottom-left of the table. Hidden when table is loading and no
   * rows of data are present. */
  renderTableHelpText?: () => JSX.Element | null;
  renderPagination?: () => JSX.Element | null;
  setExportRows?: (rows: Row[]) => void;
  onClearSelection?: () => void;
}

interface IHeaderGroup extends HeaderGroup {
  title?: string;
}

// This data table uses react-table for implementation. The relevant v7 documentation of the library
// can be found here https://react-table-v7-docs.netlify.app/docs/api/usetable

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
  keyboardSelectableRows,
  showMarkAllPages,
  isAllPagesSelected,
  toggleAllPagesSelected,
  resultsTitle = "results",
  defaultPageSize,
  defaultPageIndex,
  defaultSelectedRows = {},
  autoResetPage = true,
  primarySelectAction,
  secondarySelectActions,
  isClientSidePagination,
  onClientSidePaginationChange,
  isClientSideFilter,
  disableHighlightOnHover,
  searchQuery,
  searchQueryColumn,
  selectedDropdownFilter,
  persistSelectedRows = false,
  hideFooter = false,
  onSelectSingleRow,
  onClickRow,
  onResultsCountChange,
  renderTableHelpText,
  renderPagination,
  setExportRows,
  onClearSelection = noop,
}: IDataTableProps): JSX.Element => {
  // used to track the initial mount of the component.
  const isInitialRender = useRef(true);

  const { isOnlyObserver } = useContext(AppContext);

  const columns = useMemo(() => {
    return tableColumns;
  }, [tableColumns]);

  // The table data needs to be ordered by the order we received from the API.
  const data = useMemo(() => {
    return tableData;
  }, [tableData]);

  const initialSortBy = useMemo(() => {
    return [{ id: sortHeader, desc: sortDirection === "desc" }];
  }, [sortHeader, sortDirection]);

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
    gotoPage,
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
        sortBy: initialSortBy,
        pageIndex: defaultPageIndex,
        selectedRowIds: defaultSelectedRows,
      },
      disableMultiSort: true,
      disableSortRemove: true,
      manualSortBy,
      autoResetPage,
      // Resets row selection on (server-side) pagination
      autoResetSelectedRows: true,
      // Expands the enumerated `filterTypes` for react-table
      // (see https://github.com/TanStack/react-table/blob/alpha/packages/react-table/src/filterTypes.ts)
      // with custom `filterTypes` defined for this `useTable` instance
      filterTypes: useMemo(
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
      sortTypes: useMemo(
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
          hostPolicyStatus: (
            a: { values: Record<string, unknown[]> },
            b: { values: Record<string, unknown[]> },
            id: string
          ) => sort.hostPolicyStatus(a.values[id], b.values[id]),
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

  const { sortBy, selectedRowIds, pageIndex } = tableState;

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
    if (!isInitialRender.current && isClientSideFilter && searchQueryColumn) {
      setDebouncedClientFilter(searchQueryColumn, searchQuery || "");
    }

    // we only want to reset the selected rows if we are not persisting them
    // across table data filters
    if (!isInitialRender.current && !persistSelectedRows) {
      toggleAllRowsSelected(false); // Resets row selection on query change (client-side)
    }
    isInitialRender.current = false;
  }, [searchQuery, searchQueryColumn]);

  useEffect(() => {
    if (isClientSideFilter && selectedDropdownFilter) {
      toggleAllRowsSelected(false); // Resets row selection on filter change (client-side)
      selectedDropdownFilter === "all"
        ? setDebouncedClientFilter("platforms", "")
        : setDebouncedClientFilter("platforms", selectedDropdownFilter);
    }
  }, [selectedDropdownFilter]);

  // track previous sort state
  const prevSort = useRef<{ id?: string; desc?: boolean }>({
    id: undefined,
    desc: undefined, // desc as in descending
  });

  // This is used to listen for changes to sort. If there is a change
  // Then the sortHandler change is fired.
  useEffect(() => {
    const column = sortBy[0];
    const prev = prevSort.current;
    const newId = column?.id;
    const newDesc = column?.desc;

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

    // Only reset to page 0 if sort column/direction actually changes
    // Prevents unnecessary page resets when a column ordering changes
    // e.g. when clicking on an action that modifies the data
    const hasSortChanged =
      (!prev && (newId || newDesc !== undefined)) ||
      (prev && (prev.id !== newId || prev.desc !== newDesc));

    if (isClientSidePagination && hasSortChanged) {
      gotoPage(0); // Just this, no defaultPageIndex/etc!
    }
    prevSort.current = column
      ? { id: newId, desc: newDesc }
      : { id: undefined, desc: undefined };
  }, [sortBy, sortHeader, onSort, sortDirection, isClientSidePagination]);

  /** For clientside pagination only:
   * Whenever defaultPageIndex (the value from props, e.g. queryParams.page) changes,
   * ensure we call gotoPage so react-table reflects the correct visible page.
   */
  useEffect(() => {
    if (
      isClientSidePagination &&
      typeof defaultPageIndex === "number" &&
      pageIndex !== defaultPageIndex
    ) {
      gotoPage(defaultPageIndex);
    }
  }, [isClientSidePagination, defaultPageIndex, gotoPage, pageIndex]);

  useEffect(() => {
    if (isAllPagesSelected) {
      toggleAllRowsSelected(true);
    }
  }, [isAllPagesSelected, toggleAllRowsSelected]);

  useEffect(() => {
    setPageSize(defaultPageSize);
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
    onClearSelection();
    toggleAllRowsSelected?.(false);
    toggleAllPagesSelected?.(false);
  }, [onClearSelection, toggleAllPagesSelected, toggleAllRowsSelected]);

  const onSelectRowClick = useCallback(
    (row: any) => {
      if (disableMultiRowSelect) {
        row.toggleRowSelected();
        onSelectSingleRow && onSelectSingleRow(row);
        toggleAllRowsSelected(false);
      }
    },
    [disableMultiRowSelect, onSelectSingleRow, toggleAllRowsSelected]
  );

  const renderColumnHeader = (column: IHeaderGroup) => {
    return (
      <div className="column-header">
        {column.render("Header")}
        {column.Filter && column.render("Filter")}
      </div>
    );
  };

  const renderSelectedCount = (): JSX.Element => {
    const selectedCount = Object.entries(selectedRowIds).filter(
      ([, value]) => value
    ).length;
    return (
      <p>
        <span>
          {selectedCount}
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
    const key = kebabCase(actionButtonProps.name);
    return (
      <div className={`${baseClass}__${key}`}>
        <ActionButton {...{ key, ...actionButtonProps }} />
      </div>
    );
  };

  const renderPrimarySelectAction = (): JSX.Element | null => {
    const targetIds = selectedFlatRows.map((row: any) => row.original.id);
    const buttonText =
      typeof primarySelectAction?.buttonText === "function"
        ? primarySelectAction?.buttonText(targetIds)
        : primarySelectAction?.buttonText;
    const name = buttonText ? kebabCase(buttonText) : "primary-select-action";

    const actionProps = {
      name,
      buttonText: buttonText || "",
      onClick: primarySelectAction?.onClick || noop,
      targetIds,
      variant: primarySelectAction?.variant,
      iconSvg: primarySelectAction?.iconSvg,
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

  const tableStyles = classnames({
    "data-table__table": true,
    "data-table__no-rows": !rows.length,
    "is-observer": isOnlyObserver,
  });

  return (
    <div className={baseClass}>
      {isLoading && (
        <div className="loading-overlay">
          <Spinner />
        </div>
      )}
      <div className="data-table data-table__wrapper">
        <table className={tableStyles}>
          {Object.keys(selectedRowIds).length !== 0 && (
            <thead className="active-selection">
              <tr {...headerGroups[0].getHeaderGroupProps()}>
                <th
                  className="active-selection__checkbox"
                  {...headerGroups[0].headers[0].getHeaderProps(
                    headerGroups[0].headers[0].getSortByToggleProps({
                      title: null,
                    })
                  )}
                >
                  {headerGroups[0].headers[0].render("Header")}
                </th>
                <th className="active-selection__container">
                  <div className="active-selection__inner">
                    {renderSelectedCount()}
                    <div className="active-selection__inner-left">
                      {secondarySelectActions && renderSecondarySelectActions()}
                    </div>
                    <div className="active-selection__inner-right">
                      {primarySelectAction && renderPrimarySelectAction()}
                    </div>
                    {toggleAllPagesSelected && renderAreAllSelected()}
                    {shouldRenderToggleAllPages && (
                      <Button
                        onClick={onToggleAllPagesClick}
                        variant="text-link"
                        className="light-text"
                      >
                        <>Select all matching {resultsTitle}</>
                      </Button>
                    )}
                    <Button onClick={onClearSelectionClick} variant="text-link">
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
                  return (
                    <th
                      className={column.id ? `${column.id}__header` : ""}
                      {...column.getHeaderProps(
                        column.getSortByToggleProps({ title: null })
                      )}
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
                "clickable-row": !!onClickRow,
              });
              return (
                <tr
                  className={rowStyles}
                  {...row.getRowProps({
                    // @ts-ignore // TS complains about prop not existing
                    onClick: () => {
                      (onSelectRowClick &&
                        disableMultiRowSelect &&
                        onSelectRowClick(row)) ||
                        (disableMultiRowSelect &&
                          onClickRow &&
                          onClickRow(row));
                    },
                    // For accessibility when tabable
                    onKeyDown: (e: KeyboardEvent) => {
                      if (e.key === "Enter") {
                        e.stopPropagation();
                        (onSelectRowClick &&
                          disableMultiRowSelect &&
                          onSelectRowClick(row)) ||
                          (disableMultiRowSelect &&
                            onClickRow &&
                            onClickRow(row));
                      }
                    },
                  })}
                  // Can tab onto an entire row if a child element does not have the same onClick functionality as clicking the whole row
                  tabIndex={keyboardSelectableRows ? 0 : -1}
                >
                  {row.cells.map((cell: any, index: number) => {
                    // Only allow row click behavior on first cell
                    // if the first cell is not a checkbox
                    const cellProps = cell.getCellProps();
                    const multiRowSelectEnabled = !disableMultiRowSelect;

                    return (
                      <td
                        key={cell.column.id}
                        className={
                          cell.column.id ? `${cell.column.id}__cell` : ""
                        }
                        style={
                          multiRowSelectEnabled ? { cursor: "initial" } : {}
                        }
                        {...cellProps}
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
      {!hideFooter && (
        <div className={`${baseClass}__footer`}>
          {renderTableHelpText && !!rows?.length && (
            <div className={`${baseClass}__table-help-text`}>
              {renderTableHelpText()}
            </div>
          )}
          {isClientSidePagination ? (
            <Pagination
              disablePrev={!canPreviousPage}
              disableNext={!canNextPage}
              onPrevPage={() => {
                toggleAllRowsSelected(false); // Resets row selection on pagination (client-side)
                onClientSidePaginationChange &&
                  onClientSidePaginationChange(pageIndex - 1);
                previousPage();
              }}
              onNextPage={() => {
                toggleAllRowsSelected(false); // Resets row selection on pagination (client-side)
                onClientSidePaginationChange &&
                  onClientSidePaginationChange(pageIndex + 1);
                nextPage();
              }}
              hidePagination={!canPreviousPage && !canNextPage}
            />
          ) : (
            renderPagination && renderPagination()
          )}
        </div>
      )}
    </div>
  );
};

export default DataTable;
