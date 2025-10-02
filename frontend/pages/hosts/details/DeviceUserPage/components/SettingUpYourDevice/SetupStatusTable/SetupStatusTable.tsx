import React from "react";

import { ISetupStep } from "interfaces/setup";

import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";

import generateColumnConfigs from "./SetupStatusTableConfig";

const baseClass = "setup-status-table";

interface ISetupStatusTableProps {
  statuses: ISetupStep[];
}

const SetupStatusTable = ({ statuses }: ISetupStatusTableProps) => {
  const columnConfigs = generateColumnConfigs();

  return (
    <div className={baseClass}>
      <TableContainer
        columnConfigs={columnConfigs}
        data={statuses}
        isLoading={false}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        disableTableHeader={false}
        disablePagination
        pageSize={statuses.length}
        emptyComponent={() => (
          // will never be empty
          <EmptyTable
            header="No setup steps to complete"
            info="Setup items will appear here"
          />
        )}
      />
    </div>
  );
};

export default SetupStatusTable;
