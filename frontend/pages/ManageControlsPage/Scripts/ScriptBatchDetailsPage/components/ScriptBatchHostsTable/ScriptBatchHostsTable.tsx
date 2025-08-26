import React, { useCallback } from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";

import scriptsAPI, {
  IScriptBatchHostResultsResponse,
  IScriptBatchHostResultsQueryKey,
  ScriptBatchHostsOrderKey,
} from "services/entities/scripts";
import { OrderDirection } from "services/entities/common";

import { ScriptBatchHostStatus } from "interfaces/script";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import TableContainer from "components/TableContainer";
import DataError from "components/DataError";
import Spinner from "components/Spinner";

import generateColumnConfigs from "./ScriptBatchHostsTableConfig";

export const DEFAULT_SORT_DIRECTION = "asc";
export const DEFAULT_PAGE_SIZE = 20;
export const DEFAULT_SORT_COLUMN = "display_name";

const baseClass = "script-batch-hosts-table";

interface IScriptBatchHostsTableProps {
  batchExecutionId: string;
  hostStatus: ScriptBatchHostStatus;
  page: number;
  orderDirection: OrderDirection;
  orderKey: ScriptBatchHostsOrderKey;
}

const ScriptBatchHostsTable = ({
  batchExecutionId,
  hostStatus,
  page,
  orderDirection,
  orderKey,
}: IScriptBatchHostsTableProps) => {
  const perPage = DEFAULT_PAGE_SIZE; // TODO - allow changing this via URL?
  const { data: hostResults, isLoading, error } = useQuery<
    IScriptBatchHostResultsResponse,
    AxiosError,
    IScriptBatchHostResultsResponse,
    IScriptBatchHostResultsQueryKey[]
  >(
    [
      {
        scope: "script_batch_host_results",
        batch_execution_id: batchExecutionId,
        status: hostStatus, // TODO - param name –> host_status?
        page,
        per_page: perPage,
        order_direction: orderDirection,
        order_key: orderKey,
      },
    ],
    ({ queryKey }) => scriptsAPI.getScriptBatchHostResults(queryKey[0]),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
    }
  );

  if (error) {
    return <DataError description="Could not load host results." />;
  }

  const handleRowClick = () => alert("TODO");

  const columnConfigs = generateColumnConfigs(hostStatus);
  // const tableData = generateTableData(hostResults?.hosts || [], hostStatus);

  return (
    <div className={baseClass}>
      <TableContainer
        columnConfigs={columnConfigs}
        data={hostResults?.hosts ?? []}
        isLoading={isLoading}
        defaultSortHeader={orderKey || DEFAULT_SORT_COLUMN}
        defaultSortDirection={orderDirection || DEFAULT_SORT_DIRECTION}
        pageIndex={page}
        disableNextPage={!hostResults?.meta.has_next_results}
        pageSize={perPage}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        manualSortBy
        disableTableHeader
        emptyComponent={() => <></>} // empty state handled by parent
        disableMultiRowSelect
        searchable={false}
        onClickRow={handleRowClick}
      />
    </div>
  );
};

export default ScriptBatchHostsTable;
