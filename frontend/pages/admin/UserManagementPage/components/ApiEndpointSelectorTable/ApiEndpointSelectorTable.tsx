import React, {
  useMemo,
  useState,
  useCallback,
  useRef,
  useEffect,
} from "react";
import { useQuery } from "react-query";
import { Row } from "react-table";
import { isEmpty } from "lodash";

import TableContainer from "components/TableContainer";
import TextCell from "components/TableContainer/DataTable/TextCell/TextCell";
import PillBadge from "components/PillBadge";
import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";
import InputFieldWithIcon from "components/forms/fields/InputFieldWithIcon/InputFieldWithIcon";
import DataError from "components/DataError";
import CustomLink from "components/CustomLink";
import {
  IApiEndpoint,
  IApiEndpointRef,
  endpointKey,
} from "interfaces/api_endpoint";
import apiEndpointsAPI from "services/entities/api_endpoints";

const baseClass = "endpoint-selector-table";

interface IApiEndpointRow extends IApiEndpoint {
  id: string;
}

/** Normalize path parameter names (e.g. `:id`, `:host_id` → `:_`) so search
 * ignores parameter naming differences. */
const normalizePath = (s: string) =>
  s.toLowerCase().replace(/:[a-z0-9_]+/g, ":_");

interface IApiEndpointSelectorTableProps {
  selectedEndpoints: IApiEndpointRef[];
  onSelectionChange: (endpoints: IApiEndpointRef[]) => void;
}

interface ICellProps {
  cell: { value: string };
  row: { original: IApiEndpointRow };
}

const NameCell = (cellProps: ICellProps) => {
  const { deprecated } = cellProps.row.original;
  return (
    <span className={`${baseClass}__name-cell`}>
      <TextCell value={cellProps.cell.value} className="" />
      {deprecated && (
        <PillBadge tipContent="This endpoint is deprecated and may be removed in a future version.">
          Deprecated
        </PillBadge>
      )}
    </span>
  );
};

const searchResultsTableHeaders = [
  {
    title: "Name",
    Header: "Name",
    accessor: "display_name",
    disableSortBy: true,
    Cell: NameCell,
  },
  {
    title: "Method",
    Header: "Method",
    accessor: "method",
    disableSortBy: true,
    Cell: (cellProps: ICellProps) => <code>{cellProps.cell.value}</code>,
  },
  {
    title: "Path",
    Header: "Path",
    accessor: "path",
    disableSortBy: true,
    Cell: (cellProps: ICellProps) => <code>{cellProps.cell.value}</code>,
  },
];

const generateSelectedTableHeaders = (
  handleRemove: (row: Row<IApiEndpointRow>) => void
) => [
  ...searchResultsTableHeaders,
  {
    id: "delete",
    Header: "",
    Cell: (cellProps: { row: Row<IApiEndpointRow> }) => (
      <Button onClick={() => handleRemove(cellProps.row)} variant="icon">
        <Icon name="close-filled" />
      </Button>
    ),
    disableHidden: true,
  },
];

const ApiEndpointSelectorTable = ({
  selectedEndpoints,
  onSelectionChange,
}: IApiEndpointSelectorTableProps) => {
  const [searchText, setSearchText] = useState("");
  const dropdownRef = useRef<HTMLDivElement | null>(null);

  const { data: apiEndpoints, isLoading, error } = useQuery<
    IApiEndpoint[],
    Error
  >(["api_endpoints"], () => apiEndpointsAPI.loadAll(), {
    refetchOnWindowFocus: false,
  });

  const allRows: IApiEndpointRow[] = useMemo(
    () =>
      (apiEndpoints || []).map((ep) => ({
        ...ep,
        id: endpointKey(ep),
      })),
    [apiEndpoints]
  );

  // Filter search results: match search text and exclude already-selected.
  // Path parameter names (e.g. `:id`, `:host_id`) are normalized so searching
  // "/hosts/:id/report" matches "/hosts/:host_id/report".
  const searchResults: IApiEndpointRow[] = useMemo(() => {
    if (isEmpty(searchText)) return [];
    const query = normalizePath(searchText);
    return allRows.filter((ep) => {
      if (selectedEndpoints.some((s) => endpointKey(s) === ep.id)) return false;
      return (
        ep.display_name.toLowerCase().includes(query) ||
        normalizePath(ep.path).includes(query) ||
        ep.method.toLowerCase().includes(query)
      );
    });
  }, [allRows, searchText, selectedEndpoints]);

  const selectedRows: IApiEndpointRow[] = useMemo(
    () =>
      allRows.filter((ep) =>
        selectedEndpoints.some((s) => endpointKey(s) === ep.id)
      ),
    [allRows, selectedEndpoints]
  );

  // Close dropdown when clicking outside or pressing Escape
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target as Node)
      ) {
        setSearchText("");
      }
    };
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setSearchText("");
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, []);

  const handleRowSelect = useCallback(
    (row: Row<IApiEndpointRow>) => {
      const { method, path } = row.original;
      if (
        !selectedEndpoints.some((s) => s.method === method && s.path === path)
      ) {
        onSelectionChange([...selectedEndpoints, { method, path }]);
      }
      setSearchText("");
    },
    [selectedEndpoints, onSelectionChange]
  );

  const handleRowRemove = useCallback(
    (row: Row<IApiEndpointRow>) => {
      const { method, path } = row.original;
      onSelectionChange(
        selectedEndpoints.filter((s) => s.method !== method || s.path !== path)
      );
    },
    [selectedEndpoints, onSelectionChange]
  );

  const selectedTableHeaders = useMemo(
    () => generateSelectedTableHeaders(handleRowRemove),
    [handleRowRemove]
  );

  const isDropdownOpen = !isEmpty(searchText);
  const showResults = isDropdownOpen && !error;
  const showSearchError = isDropdownOpen && !!error;

  return (
    <div className={baseClass}>
      <InputFieldWithIcon
        type="search"
        iconSvg="search"
        value={searchText}
        placeholder="Search by name or path"
        onChange={setSearchText}
      />
      <span className="form-field__help-text">
        You can find this information in the{" "}
        <CustomLink
          url="https://fleetdm.com/docs/rest-api/rest-api"
          text="REST API docs"
          newTab
        />
      </span>
      {showResults && (
        <div className={`${baseClass}__search-dropdown`} ref={dropdownRef}>
          <TableContainer<Row<IApiEndpointRow>>
            columnConfigs={searchResultsTableHeaders}
            data={searchResults}
            isLoading={isLoading}
            emptyComponent={() => (
              <div className="empty-search">
                <div className="empty-search__inner">
                  <h4>No matching API endpoints.</h4>
                  <p>
                    Please check the API documentation and try again.
                    <br />
                    Experimental endpoints are not supported.
                  </p>
                </div>
              </div>
            )}
            showMarkAllPages={false}
            isAllPagesSelected={false}
            disableCount
            disableMultiRowSelect
            isClientSidePagination
            pageSize={10}
            onClickRow={handleRowSelect}
          />
        </div>
      )}
      {showSearchError && (
        <div className={`${baseClass}__search-dropdown`} ref={dropdownRef}>
          <DataError />
        </div>
      )}
      {selectedRows.length > 0 && (
        <div className={`${baseClass}__selected-table`}>
          <TableContainer
            columnConfigs={selectedTableHeaders}
            data={selectedRows}
            isLoading={false}
            showMarkAllPages={false}
            isAllPagesSelected={false}
            disableCount
            disablePagination
            emptyComponent={() => <></>}
          />
        </div>
      )}
    </div>
  );
};

export default ApiEndpointSelectorTable;
