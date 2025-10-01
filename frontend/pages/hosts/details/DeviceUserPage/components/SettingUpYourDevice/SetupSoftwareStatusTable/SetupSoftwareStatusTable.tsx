import React from "react";

import { ISetupSoftwareStatus } from "interfaces/software";

import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";

import generateColumnConfigs from "./SetupSoftwareStatusTableConfig";

const baseClass = "setup-software-status-table";

interface ISetupSoftwareStatusTableProps {
  statuses: ISetupSoftwareStatus[];
}

const SetupSoftwareStatusTable = ({
  statuses,
}: ISetupSoftwareStatusTableProps) => {
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
            header="No software to install"
            info="Software setup status will appear here"
          />
        )}
      />
    </div>
  );
};

export default SetupSoftwareStatusTable;
