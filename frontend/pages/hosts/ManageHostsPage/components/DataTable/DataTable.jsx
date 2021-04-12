import React, {
  useMemo,
  useEffect,
  useRef,
  useCallback,
  useState,
} from "react";
import PropTypes from "prop-types";
import {
  useTable,
  useGlobalFilter,
  useSortBy,
  useAsyncDebounce,
} from "react-table";
import { useSelector, useDispatch } from "react-redux";

import Spinner from "components/loaders/Spinner";
import Pagination from "components/Pagination";
import scrollToTop from "utilities/scroll_to_top";

const DEFAULT_PAGE_INDEX = 0;
const DEBOUNCE_QUERY_DELAY = 300;
const DEFAULT_RESULTS_NAME = "results";

const baseClass = "data-table-container";

const generateResultsCountText = (
  name = DEFAULT_RESULTS_NAME,
  pageIndex,
  itemsPerPage,
  resultsCount
) => {
  if (itemsPerPage === resultsCount) return `${itemsPerPage}+ ${name}`;

  if (pageIndex !== 0 && resultsCount <= itemsPerPage)
    return `${itemsPerPage}+ ${name}`;

  return `${resultsCount} ${name}`;
};

// This data table uses react-table for implementation. The relevant documentation of the library
// can be found here https://react-table.tanstack.com/docs/api/useTable
const DataTable = (props) => {
  const {
    // selectedFilter is passed from parent, as it ultimately comes from the router and this
    // component cannot access the router state.
    selectedFilter,
    searchQuery,
    hiddenColumns,
    tableColumns,
    pageSize,
    defaultSortHeader,
    resultsName,
    fetchDataAction,
    entity,
    emptyComponent,
  } = props;

  const [pageIndex, setPageIndex] = useState(DEFAULT_PAGE_INDEX);

  const dispatch = useDispatch();
  const loadingEntity = useSelector((state) => state.entities[entity].loading);
  const entityData = useSelector((state) => state.entities[entity].data);
  const apiOrder = useSelector((state) => state.entities[entity].originalOrder);

  // This variable is used to keep the react-table state persistent across server calls for new data.
  // You can read more about this here technique here:
  // https://react-table.tanstack.com/docs/faq#how-do-i-stop-my-table-state-from-automatically-resetting-when-my-data-changes
  const skipPageResetRef = useRef();

  const pageIndexChangeRef = useRef();

  const columns = useMemo(() => {
    return tableColumns;
  }, [tableColumns]);

  // The table data needs to be ordered by the order we received from the API.
  const data = useMemo(() => {
    return apiOrder.map((id) => {
      return entityData[id];
    });
  }, [entityData, apiOrder]);

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
        sortBy: [{ id: defaultSortHeader, desc: true }],
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
    useSortBy
  );
  const { globalFilter, sortBy } = tableState;

  const debouncedGlobalFilter = useAsyncDebounce((value) => {
    skipPageResetRef.current = true;
    setGlobalFilter(value || undefined);
  }, DEBOUNCE_QUERY_DELAY);

  const onPaginationChange = useCallback(
    (newPage) => {
      if (newPage > pageIndex) {
        setPageIndex(pageIndex + 1);
      } else {
        setPageIndex(pageIndex - 1);
      }
      pageIndexChangeRef.current = true;
      scrollToTop();
    },
    [pageIndex, setPageIndex]
  );

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
  // entity data.
  useEffect(() => {
    if (pageIndexChangeRef.current) {
      // the pageIndex has changed
      dispatch(
        fetchDataAction(
          pageIndex,
          pageSize,
          selectedFilter,
          globalFilter,
          sortBy
        )
      );
    } else {
      // something besides pageIndex changed. we want to get results starting at the first page
      // NOTE: currently this causes the request to fire twice if the user is not on the first page
      // of results. Need to come back to this and figure out how to get it to
      // only fire once.
      setPageIndex(0);
      dispatch(
        fetchDataAction(0, pageSize, selectedFilter, globalFilter, sortBy)
      );
    }
    skipPageResetRef.current = false;
    pageIndexChangeRef.current = false;
  }, [
    fetchDataAction,
    dispatch,
    pageIndex,
    pageSize,
    selectedFilter,
    globalFilter,
    sortBy,
  ]);

  // No entities for this result.
  if (!loadingEntity && Object.values(entityData).length === 0) {
    const NoResultsComponent = emptyComponent;
    return <NoResultsComponent />;
  }

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__topper`}>
        <p className={`${baseClass}__results-count`}>
          {generateResultsCountText(
            resultsName,
            pageIndex,
            pageSize,
            rows.length
          )}
        </p>
      </div>
      <div className={"data-table data-table__wrapper"}>
        {loadingEntity && (
          <div className={"loading-overlay"}>
            <Spinner />
          </div>
        )}
        <table className={"data-table__table"}>
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

      <Pagination
        resultsOnCurrentPage={rows.length}
        currentPage={pageIndex}
        resultsPerPage={pageSize}
        onPaginationChange={onPaginationChange}
      />
    </div>
  );
};

DataTable.propTypes = {
  selectedFilter: PropTypes.string,
  searchQuery: PropTypes.string,
  tableColumns: PropTypes.arrayOf(PropTypes.object), // TODO: create proper interface for this
  hiddenColumns: PropTypes.arrayOf(PropTypes.string),
  pageSize: PropTypes.number,
  defaultSortHeader: PropTypes.string,
  resultsName: PropTypes.string,
  fetchDataAction: PropTypes.func,
  entity: PropTypes.string,
  emptyComponent: PropTypes.func,
};

export default DataTable;
