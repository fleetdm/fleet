import React from "react";
import TableContainer from "components/TableContainer";
import { IMacSettings } from "interfaces/mdm";

import tableHeaders from "./MacSettingsTableConfig";

const baseClass = "macsettings-table";

interface IMacSettingsTableProps {
  hostMacSettings?: IMacSettings;
}

const MacSettingsTable = ({ hostMacSettings }: IMacSettingsTableProps) => {
  return (
    <div className={baseClass}>
      <TableContainer
        resultsTitle="settings"
        defaultSortHeader="name"
        columns={tableHeaders}
        data={hostMacSettings}
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
