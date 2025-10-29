import React, { useMemo } from "react";
import TableContainer from "components/TableContainer";

import generateTableHeaders, {
  IHostMdmProfileWithAddedStatus,
} from "./OSSettingsTableConfig";

const baseClass = "os-settings-table";

interface IOSSettingsTableProps {
  canResendProfiles: boolean;
  tableData: IHostMdmProfileWithAddedStatus[];
  resendRequest: (profileUUID: string) => Promise<void>;
  onProfileResent: () => void;
}

const OSSettingsTable = ({
  canResendProfiles,
  tableData,
  resendRequest,
  onProfileResent,
}: IOSSettingsTableProps) => {
  // useMemo prevents tooltip flashing during host data refetch
  const tableConfig = useMemo(
    () =>
      generateTableHeaders(canResendProfiles, resendRequest, onProfileResent),
    [canResendProfiles, resendRequest, onProfileResent]
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
