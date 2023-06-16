import React from "react";
import TableContainer from "components/TableContainer";

import tableHeaders, { IMacSettingsTableRow } from "./MacSettingsTableConfig";

const baseClass = "macsettings-table";

interface IMacSettingsTableProps {
  tableData?: IMacSettingsTableRow[];
}

const MacSettingsTable = ({ tableData }: IMacSettingsTableProps) => {
  return (
    <div className={baseClass}>
      <TableContainer
        resultsTitle="settings"
        defaultSortHeader="name"
        columns={tableHeaders}
        data={tableData}
        emptyComponent={"symbol"}
        isLoading={false}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        disablePagination
      />
    </div>
  );
};

export default MacSettingsTable;
