import React, { useMemo } from "react";
import TableContainer from "components/TableContainer";

import generateTableHeaders, {
  IHostMdmProfileWithAddedStatus,
} from "./OSSettingsTableConfig";

const baseClass = "os-settings-table";

interface IOSSettingsTableProps {
  canResendProfiles: boolean;
  canRotateRecoveryLockPassword?: boolean;
  canResendHostNameTemplate?: boolean;
  tableData: IHostMdmProfileWithAddedStatus[];
  resendRequest: (profileUUID: string) => Promise<void>;
  resendCertificateRequest?: (certificateTemplateId: number) => Promise<void>;
  rotateRecoveryLockPassword?: () => Promise<void>;
  resendHostNameTemplate?: () => Promise<void>;
  onProfileResent: () => void;
}

const OSSettingsTable = ({
  canResendProfiles,
  canRotateRecoveryLockPassword = false,
  canResendHostNameTemplate = false,
  tableData,
  resendRequest,
  resendCertificateRequest,
  rotateRecoveryLockPassword,
  resendHostNameTemplate,
  onProfileResent,
}: IOSSettingsTableProps) => {
  // useMemo prevents tooltip flashing during host data refetch
  const tableConfig = useMemo(
    () =>
      generateTableHeaders(
        canResendProfiles,
        resendRequest,
        onProfileResent,
        resendCertificateRequest,
        canRotateRecoveryLockPassword,
        rotateRecoveryLockPassword,
        canResendHostNameTemplate,
        resendHostNameTemplate
      ),
    [
      canResendProfiles,
      resendRequest,
      onProfileResent,
      canRotateRecoveryLockPassword,
      rotateRecoveryLockPassword,
      canResendHostNameTemplate,
      resendHostNameTemplate,
      resendCertificateRequest,
    ]
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
