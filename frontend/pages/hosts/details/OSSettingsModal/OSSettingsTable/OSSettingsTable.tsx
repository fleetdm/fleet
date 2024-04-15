import React from "react";
import TableContainer from "components/TableContainer";

import generateTableHeaders, {
  IHostMdmProfileWithAddedStatus,
} from "./OSSettingsTableConfig";

const baseClass = "os-settings-table";

interface IOSSettingsTableProps {
  canResendProfiles: boolean;
  hostId: number;
  tableData: IHostMdmProfileWithAddedStatus[];
  onProfileResent?: () => void;
}

const OSSettingsTable = ({
  canResendProfiles,
  hostId,
  tableData,
  onProfileResent,
}: IOSSettingsTableProps) => {
  const tableConfig = generateTableHeaders(
    hostId,
    canResendProfiles,
    onProfileResent
  );

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
