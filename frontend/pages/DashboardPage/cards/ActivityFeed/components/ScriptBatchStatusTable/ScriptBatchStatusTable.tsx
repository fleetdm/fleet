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
  onClickCancel: () => void;
}

const ScriptBatchStatusTable = ({
  statusData,
  onClickCancel,
}: IScriptBatchStatusTableProps) => {
  const columnConfigs = useMemo(() => {
    return generateTableConfig(onClickCancel);
  }, [onClickCancel]);
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
