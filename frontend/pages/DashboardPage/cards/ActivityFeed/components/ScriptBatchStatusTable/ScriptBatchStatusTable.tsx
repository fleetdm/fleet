import EmptyTable from "components/EmptyTable";
import TableContainer from "components/TableContainer";
import React, { useMemo } from "react";
import { IScriptBatchSummaryResponse } from "services/entities/scripts";
import {
  generateTableConfig,
  generateTableData,
} from "./ScriptBatchStatusTableConfig";

const baseClass = "script-batch-status-table";

interface IScriptBatchStatusTableProps {
  statusData: IScriptBatchSummaryResponse;
  batchExecutionId: string;
  onClickCancel: () => void;
}

const ScriptBatchStatusTable = ({
  statusData,
  batchExecutionId,
  onClickCancel,
}: IScriptBatchStatusTableProps) => {
  const columnConfigs = useMemo(() => {
    return generateTableConfig(
      batchExecutionId,
      onClickCancel,
      statusData.team_id
    );
  }, [batchExecutionId, onClickCancel, statusData.team_id]);
  const tableData = generateTableData(statusData);

  return (
    <TableContainer
      className={baseClass}
      columnConfigs={columnConfigs}
      data={tableData}
      isLoading={false}
      emptyComponent={() => <EmptyTable />}
      showMarkAllPages={false}
      isAllPagesSelected={false}
      manualSortBy
      disableTableHeader
      disablePagination
      disableCount
      hideFooter
    />
  );
};

export default ScriptBatchStatusTable;
