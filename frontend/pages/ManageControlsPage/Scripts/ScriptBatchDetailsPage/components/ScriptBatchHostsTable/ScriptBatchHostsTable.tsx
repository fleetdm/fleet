import React, { useCallback } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";
import { AxiosError } from "axios";

import PATHS from "router/paths";

import scriptsAPI, {
  IScriptBatchHostResultsResponse,
  IScriptBatchHostResultsQueryKey,
  ScriptBatchHostsOrderKey,
} from "services/entities/scripts";
import { OrderDirection } from "services/entities/common";

import {
  SCRIPT_BATCH_HOST_EXECUTED_STATUSES,
  ScriptBatchHostStatus,
} from "interfaces/script";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { getNextLocationPath } from "utilities/helpers";

import TableContainer from "components/TableContainer";
import DataError from "components/DataError";
import { ITableQueryData } from "components/TableContainer/TableContainer";

import generateColumnConfigs from "./ScriptBatchHostsTableConfig";

export const DEFAULT_SORT_DIRECTION = "asc";
export const DEFAULT_PAGE_SIZE = 20;
export const DEFAULT_SORT_COLUMN = "display_name";

const baseClass = "script-batch-hosts-table";

interface IScriptBatchHostsTableProps {
  batchExecutionId: string;
  selectedHostStatus: ScriptBatchHostStatus;
  page: number;
  orderDirection: OrderDirection;
  orderKey: ScriptBatchHostsOrderKey;
  setHostScriptExecutionIdForModal: (id: string) => void;
  router: InjectedRouter;
}

const ScriptBatchHostsTable = ({
  batchExecutionId,
  selectedHostStatus,
  page,
  orderDirection,
  orderKey,
  setHostScriptExecutionIdForModal,
  router,
}: IScriptBatchHostsTableProps) => {
  const perPage = DEFAULT_PAGE_SIZE;
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
        status: selectedHostStatus,
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

  const handleRowClick = useCallback(
    (row: any) => {
      if (SCRIPT_BATCH_HOST_EXECUTED_STATUSES.includes(selectedHostStatus)) {
        setHostScriptExecutionIdForModal(row.original.script_execution_id);
      } else {
        router.push(PATHS.HOST_DETAILS(row.original.id));
      }
    },
    [router, selectedHostStatus, setHostScriptExecutionIdForModal]
  );

  const handleQueryChange = useCallback(
    (newTableQuery: ITableQueryData) => {
      const {
        pageIndex: newPageIndex,
        sortDirection: newOrderDirection,
        sortHeader: newOrderKey,
      } = newTableQuery;

      const newQueryParams: { [key: string]: string | number | undefined } = {};
      newQueryParams.status = selectedHostStatus;
      newQueryParams.order_key = newOrderKey;
      newQueryParams.order_direction = newOrderDirection;
      newQueryParams.page = newPageIndex.toString();

      if (newOrderKey !== orderKey || newOrderDirection !== orderDirection) {
        newQueryParams.page = "0";
      }
      const path = getNextLocationPath({
        pathPrefix: PATHS.CONTROLS_SCRIPTS_BATCH_DETAILS(batchExecutionId),
        queryParams: newQueryParams,
      });

      // replace instead of push here keeps browser history clear and allows cleaner forward/back navigation
      router.replace(path);
    },
    [selectedHostStatus, orderKey, orderDirection, batchExecutionId, router]
  );

  if (error) {
    return <DataError description="Could not load host results." />;
  }

  const columnConfigs = generateColumnConfigs(selectedHostStatus);

  return (
    <div className={baseClass}>
      <TableContainer
        columnConfigs={columnConfigs}
        data={hostResults?.hosts ?? []}
        isLoading={isLoading}
        defaultSortHeader={orderKey || DEFAULT_SORT_COLUMN}
        defaultSortDirection={orderDirection || DEFAULT_SORT_DIRECTION}
        pageIndex={page}
        pageSize={perPage}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        manualSortBy
        disableTableHeader
        emptyComponent={() => <></>} // empty state handled by parent
        disableMultiRowSelect
        searchable={false}
        onClickRow={handleRowClick}
        onQueryChange={handleQueryChange}
      />
    </div>
  );
};

export default ScriptBatchHostsTable;
