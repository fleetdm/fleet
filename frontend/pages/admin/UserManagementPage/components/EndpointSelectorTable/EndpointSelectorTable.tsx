import React, { useMemo, useState, useCallback } from "react";
import { useQuery } from "react-query";
import { Row } from "react-table";

import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import TextCell from "components/TableContainer/DataTable/TextCell/TextCell";
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

const tableHeaders = [
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
];

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

  const handleRowSelect = useCallback(
    (row: Row<IEndpointRow>) => {
      const key = row.original.id;
      if (selectedEndpointKeys.includes(key)) {
        onSelectionChange(selectedEndpointKeys.filter((k) => k !== key));
      } else {
        onSelectionChange([...selectedEndpointKeys, key]);
      }
    },
    [selectedEndpointKeys, onSelectionChange]
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
      <TableContainer<Row<IEndpointRow>>
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
        onClickRow={handleRowSelect}
        pageSize={PAGE_SIZE}
        disableCount
      />
    </div>
  );
};

export default EndpointSelectorTable;
