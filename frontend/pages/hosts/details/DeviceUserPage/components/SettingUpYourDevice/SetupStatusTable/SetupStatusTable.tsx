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

  // Sort the statuses so that scripts are always at the bottom.
  statuses.sort((a, b) => {
    if (a.type === b.type) {
      return 0;
    }
    if (a.type === "script") {
      return 1;
    }
    return -1;
  });

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
        manualSortBy
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
