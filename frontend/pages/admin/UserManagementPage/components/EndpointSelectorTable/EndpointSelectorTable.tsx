import React, { useMemo, useState, useCallback } from "react";
import { useQuery } from "react-query";

import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import TextCell from "components/TableContainer/DataTable/TextCell/TextCell";
import Checkbox from "components/forms/fields/Checkbox";
import PillBadge from "components/PillBadge";
import EmptyTable from "components/EmptyTable";
import { IApiEndpoint } from "interfaces/api_endpoint";
import apiEndpointsAPI from "services/entities/api_endpoints";

const baseClass = "endpoint-selector-table";

const PAGE_SIZE = 10;

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

const EmptyEndpointsTable = () => (
  <EmptyTable
    header="No matching API endpoints"
    info={
      <>
        Please check the API documentation and try again.
        <br />
        Experimental endpoints are not supported.
      </>
    }
  />
);

const EndpointSelectorTable = ({
  selectedEndpointKeys,
  onSelectionChange,
}: IEndpointSelectorTableProps) => {
  const [searchQuery, setSearchQuery] = useState("");

  const { data: apiEndpoints, isLoading, error } = useQuery<
    IApiEndpoint[],
    Error
  >(["api_endpoints"], () => apiEndpointsAPI.loadAll(), {
    refetchOnWindowFocus: false,
  });

  const tableData: IEndpointRow[] = useMemo(
    () =>
      (apiEndpoints || []).map((ep) => ({
        ...ep,
        id: endpointKey(ep),
      })),
    [apiEndpoints]
  );

  // Filtered row keys for the header "select all" checkbox
  const filteredRowKeys = useMemo(() => {
    if (!searchQuery) return tableData.map((ep) => ep.id);
    const query = searchQuery.toLowerCase();
    return tableData
      .filter(
        (ep) =>
          ep.display_name.toLowerCase().includes(query) ||
          ep.path.toLowerCase().includes(query)
      )
      .map((ep) => ep.id);
  }, [tableData, searchQuery]);

  const allFilteredSelected =
    filteredRowKeys.length > 0 &&
    filteredRowKeys.every((key) => selectedEndpointKeys.includes(key));

  const someFilteredSelected =
    !allFilteredSelected &&
    filteredRowKeys.some((key) => selectedEndpointKeys.includes(key));

  const toggleEndpoint = useCallback(
    (key: string) => {
      if (selectedEndpointKeys.includes(key)) {
        onSelectionChange(selectedEndpointKeys.filter((k) => k !== key));
      } else {
        onSelectionChange([...selectedEndpointKeys, key]);
      }
    },
    [selectedEndpointKeys, onSelectionChange]
  );

  const toggleAllFiltered = useCallback(() => {
    if (allFilteredSelected) {
      onSelectionChange(
        selectedEndpointKeys.filter((k) => !filteredRowKeys.includes(k))
      );
    } else {
      const newKeys = new Set([...selectedEndpointKeys, ...filteredRowKeys]);
      onSelectionChange(Array.from(newKeys));
    }
  }, [
    allFilteredSelected,
    selectedEndpointKeys,
    filteredRowKeys,
    onSelectionChange,
  ]);

  const tableHeaders = useMemo(
    () => [
      {
        title: "",
        Header: () => (
          <Checkbox
            value={allFilteredSelected}
            indeterminate={someFilteredSelected}
            name="select-all-endpoints"
            onChange={toggleAllFiltered}
          />
        ),
        accessor: "id",
        disableSortBy: true,
        Cell: (cellProps: ICellProps) => (
          <Checkbox
            value={selectedEndpointKeys.includes(cellProps.cell.value)}
            name={`endpoint-${cellProps.cell.value}`}
            onChange={() => toggleEndpoint(cellProps.cell.value)}
          />
        ),
      },
      {
        title: "Name",
        Header: "Name",
        accessor: "display_name",
        disableSortBy: true,
        Cell: (cellProps: ICellProps) => {
          const { deprecated } = cellProps.row.original;
          return (
            <span>
              <TextCell value={cellProps.cell.value} />
              {deprecated && (
                <PillBadge tipContent="This endpoint is deprecated and may be removed in a future version.">
                  Deprecated
                </PillBadge>
              )}
            </span>
          );
        },
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
    ],
    [
      selectedEndpointKeys,
      allFilteredSelected,
      someFilteredSelected,
      toggleEndpoint,
      toggleAllFiltered,
    ]
  );

  const onQueryChange = useCallback((queryData: ITableQueryData) => {
    setSearchQuery(queryData.searchQuery);
  }, []);

  if (error) {
    return (
      <EmptyTable
        header="Could not load API endpoints"
        info="Please try again later."
      />
    );
  }

  return (
    <div className={baseClass}>
      <TableContainer
        columnConfigs={tableHeaders}
        data={tableData}
        isLoading={isLoading}
        resultsTitle="endpoints"
        emptyComponent={EmptyEndpointsTable}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        disableMultiRowSelect
        isClientSidePagination
        isClientSideFilter
        searchable
        wideSearch
        inputPlaceHolder="Search by name or path"
        filters={{ global: searchQuery }}
        onQueryChange={onQueryChange}
        pageSize={PAGE_SIZE}
        disableCount
      />
    </div>
  );
};

export default EndpointSelectorTable;
