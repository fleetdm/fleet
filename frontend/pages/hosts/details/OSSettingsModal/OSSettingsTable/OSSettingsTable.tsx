import React from "react";
import TableContainer from "components/TableContainer";

import tableHeaders, {
  IHostMdmProfileWithAddedStatus,
} from "./OSSettingsTableConfig";

const baseClass = "os-settings-table";

interface IOSSettingsTableProps {
  tableData?: IHostMdmProfileWithAddedStatus[];
}

const OSSettingsTable = ({ tableData }: IOSSettingsTableProps) => {
  return (
    <div className={baseClass}>
      <TableContainer
        resultsTitle="settings"
        defaultSortHeader="name"
        columnConfigs={tableHeaders}
        data={tableData}
        emptyComponent="symbol"
        isLoading={false}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        disablePagination
      />
    </div>
  );
};

export default OSSettingsTable;
