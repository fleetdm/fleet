import React from "react";
import TableContainer from "components/TableContainer";

import generateTableHeaders, {
  IHostMdmProfileWithAddedStatus,
} from "./OSSettingsTableConfig";

const baseClass = "os-settings-table";

interface IOSSettingsTableProps {
  hostId?: number;
  tableData?: IHostMdmProfileWithAddedStatus[];
  onProfileResent?: () => void;
}

const OSSettingsTable = ({
  hostId,
  tableData,
  onProfileResent,
}: IOSSettingsTableProps) => {
  const tableConfig = generateTableHeaders(hostId, onProfileResent);

  return (
    <div className={baseClass}>
      <TableContainer
        resultsTitle="settings"
        defaultSortHeader="name"
        columnConfigs={tableConfig}
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
