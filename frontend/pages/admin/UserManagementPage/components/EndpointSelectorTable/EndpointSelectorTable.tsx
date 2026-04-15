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
import { IApiEndpoint } from "interfaces/api_endpoint";
import apiEndpointsAPI from "services/entities/api_endpoints";

const baseClass = "endpoint-selector-table";

/** Unique key for an endpoint since there's no `id` field */
const endpointKey = (ep: IApiEndpoint) => `${ep.method} ${ep.path}`;

interface IEndpointRow extends IApiEndpoint {
  id: string;
}

interface IEndpointSelectorTableProps {
  selectedEndpointKeys: string[];
  onSelectionChange: (selectedKeys: string[]) => void;
}

interface ICellProps {
  cell: { value: string };
  row: { original: IEndpointRow };
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
    title: "Protocol",
    Header: "Protocol",
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
  handleRemove: (row: Row<IEndpointRow>) => void
) => [
  ...searchResultsTableHeaders,
  {
    id: "delete",
    Header: "",
    Cell: (cellProps: { row: Row<IEndpointRow> }) => (
      <Button onClick={() => handleRemove(cellProps.row)} variant="icon">
        <Icon name="close-filled" />
      </Button>
    ),
    disableHidden: true,
  },
];

const EndpointSelectorTable = ({
  selectedEndpointKeys,
  onSelectionChange,
}: IEndpointSelectorTableProps) => {
  const [searchText, setSearchText] = useState("");
  const [isActiveSearch, setIsActiveSearch] = useState(false);
  const dropdownRef = useRef<HTMLDivElement | null>(null);

  const { data: apiEndpoints, isLoading, error } = useQuery<
    IApiEndpoint[],
    Error
  >(["api_endpoints"], () => apiEndpointsAPI.loadAll(), {
    refetchOnWindowFocus: false,
  });

  const allRows: IEndpointRow[] = useMemo(
    () =>
      (apiEndpoints || []).map((ep) => ({
        ...ep,
        id: endpointKey(ep),
      })),
    [apiEndpoints]
  );

  // Filter search results: match search text and exclude already-selected
  const searchResults: IEndpointRow[] = useMemo(() => {
    if (isEmpty(searchText)) return [];
    const query = searchText.toLowerCase();
    return allRows.filter(
      (ep) =>
        !selectedEndpointKeys.includes(ep.id) &&
        (ep.display_name.toLowerCase().includes(query) ||
          ep.path.toLowerCase().includes(query) ||
          ep.method.toLowerCase().includes(query))
    );
  }, [allRows, searchText, selectedEndpointKeys]);

  const selectedRows: IEndpointRow[] = useMemo(
    () => allRows.filter((ep) => selectedEndpointKeys.includes(ep.id)),
    [allRows, selectedEndpointKeys]
  );

  // Close dropdown when clicking outside
  useEffect(() => {
    if (!isLoading) {
      const handleClickOutside = (event: MouseEvent) => {
        if (
          dropdownRef.current &&
          !dropdownRef.current.contains(event.target as Node)
        ) {
          setIsActiveSearch(false);
        }
      };
      document.addEventListener("mousedown", handleClickOutside);
      return () => {
        document.removeEventListener("mousedown", handleClickOutside);
      };
    }
    return undefined;
  }, [isLoading]);

  // Show dropdown when there's search text
  useEffect(() => {
    setIsActiveSearch(!isEmpty(searchText) && (!error || isLoading));
  }, [searchText, error, isLoading]);

  const handleRowSelect = useCallback(
    (row: Row<IEndpointRow>) => {
      const key = row.original.id;
      if (!selectedEndpointKeys.includes(key)) {
        onSelectionChange([...selectedEndpointKeys, key]);
      }
      setSearchText("");
    },
    [selectedEndpointKeys, onSelectionChange]
  );

  const handleRowRemove = useCallback(
    (row: Row<IEndpointRow>) => {
      onSelectionChange(
        selectedEndpointKeys.filter((k) => k !== row.original.id)
      );
    },
    [selectedEndpointKeys, onSelectionChange]
  );

  const selectedTableHeaders = useMemo(
    () => generateSelectedTableHeaders(handleRowRemove),
    [handleRowRemove]
  );

  const isSearchError = !isEmpty(searchText) && !!error;

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
      {isActiveSearch && (
        <div className={`${baseClass}__search-dropdown`} ref={dropdownRef}>
          <TableContainer<Row<IEndpointRow>>
            columnConfigs={searchResultsTableHeaders}
            data={searchResults}
            isLoading={isLoading}
            emptyComponent={() => (
              <div className="empty-search">
                <div className="empty-search__inner">
                  <h4>No matching endpoints.</h4>
                  <p>Try a different search term.</p>
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
      {isSearchError && (
        <div className={`${baseClass}__search-dropdown`}>
          <DataError />
        </div>
      )}
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
    </div>
  );
};

export default EndpointSelectorTable;
