import React from "react";

import { IEnhancedSetupStep } from "interfaces/setup";

import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";

import generateColumnConfigs from "./SetupStatusTableConfig";

const baseClass = "setup-status-table";

interface ISetupStatusTableProps {
  statuses: IEnhancedSetupStep[];
}

const SetupStatusTable = ({ statuses }: ISetupStatusTableProps) => {
  const columnConfigs = generateColumnConfigs();

  // Sort the statuses so that it's status of software, then software scripts, then scripts
  const order = ["software_install", "software_script_run", "script_run"];

  statuses.sort((a: IEnhancedSetupStep, b: IEnhancedSetupStep) => {
    return order.indexOf(a.type) - order.indexOf(b.type);
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
