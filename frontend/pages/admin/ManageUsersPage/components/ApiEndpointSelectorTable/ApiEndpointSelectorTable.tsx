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

/** Split on path separators, whitespace, and word-boundary punctuation so
 * both names ("List hosts") and paths ("/api/v1/fleet/hosts") can be
 * compared word-by-word. */
const WORD_SPLIT_RE = /[\s/_-]+/;

/** Score how well a single field matches the query: exact match ranks
 * highest, then prefix match, then whole-word match, then any substring
 * match. Returns 0 when there's no match at all. */
const scoreField = (field: string, query: string): number => {
  if (!field || !query) return 0;
  if (field === query) return 100;
  if (field.startsWith(query)) return 90;
  if (field.split(WORD_SPLIT_RE).filter(Boolean).includes(query)) return 70;
  if (field.includes(query)) return 50;
  return 0;
};

/** An endpoint's relevance is the strongest match across its name and path —
 * a strong hit on one field can outrank a weak hit on the other. Method
 * matches (e.g. searching "post") are ranked below any name/path match. */
const scoreEndpoint = (ep: IApiEndpointRow, query: string): number =>
  Math.max(
    scoreField(ep.display_name.toLowerCase(), query),
    scoreField(normalizePath(ep.path), query),
    ep.method.toLowerCase().includes(query) ? 10 : 0
  );

/** Fewer path segments = a broader, higher-level endpoint. Used to break
 * score ties so e.g. `/hosts` sorts before `/hosts/:id/software`. */
const pathDepth = (path: string) => path.split("/").filter(Boolean).length;

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

  // Filter search results: match search text and exclude already-selected,
  // then rank by relevance (best match across name/path first, broader
  // paths breaking ties) rather than leaving them in catalog order.
  // Path parameter names (e.g. `:id`, `:host_id`) are normalized so searching
  // "/hosts/:id/report" matches "/hosts/:host_id/report".
  const searchResults: IApiEndpointRow[] = useMemo(() => {
    if (isEmpty(searchText)) return [];
    const query = normalizePath(searchText);
    return allRows
      .filter((ep) => !selectedEndpoints.some((s) => endpointKey(s) === ep.id))
      .map((ep) => ({ ep, score: scoreEndpoint(ep, query) }))
      .filter(({ score }) => score > 0)
      .sort(
        (a, b) =>
          b.score - a.score || pathDepth(a.ep.path) - pathDepth(b.ep.path)
      )
      .map(({ ep }) => ep);
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
          url="https://fleetdm.com/docs/api/rest-api"
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
            disablePagination
            // Without this, TableContainer's default sort (by a "name"
            // column that doesn't exist here) silently re-shuffles rows via
            // react-table's built-in sorting, discarding the relevance
            // order computed above.
            manualSortBy
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
