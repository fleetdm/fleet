import React from "react";
import TableContainer from "components/TableContainer";
import { IHostMacMdmProfile } from "interfaces/mdm";

import tableHeaders from "./MacSettingsTableConfig";

const baseClass = "macsettings-table";

interface IMacSettingsTableProps {
  hostMacSettings?: IHostMacMdmProfile[];
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
