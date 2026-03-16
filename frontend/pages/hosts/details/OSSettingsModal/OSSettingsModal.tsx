import React, { memo, useMemo } from "react";
import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { IHostMdmData } from "interfaces/host";

import OSSettingsTable from "./OSSettingsTable";
import { generateTableData } from "./OSSettingsTable/OSSettingsTableConfig";

interface IOSSettingsModalProps {
  platform: string;
  hostMDMData: IHostMdmData;
  /** controls showing the action for a user to resend a profile. Defaults to `false` */
  canResendProfiles?: boolean;
  /** controls showing the rotate action for the recovery lock password row. Defaults to `false` */
  canRotateRecoveryLockPassword?: boolean;
  /** This request method will be called when a user clicks on the resend button.
   * This behaviour is dynamic based on the page this modal is rendered on
   * so we allow the request function to be passed in */
  resendRequest: (profileUUID: string) => Promise<void>;
  rotateRecoveryLockPassword?: () => Promise<void>;
  onClose: () => void;
  /** handler that fires when a profile was reset. Requires `canResendProfiles` prop
   * to be `true`, otherwise has no effect.
   */
  onProfileResent: () => void;
}

const baseClass = "os-settings-modal";

const OSSettingsModal = ({
  platform,
  hostMDMData,
  canResendProfiles = false,
  canRotateRecoveryLockPassword = false,
  onClose,
  resendRequest,
  rotateRecoveryLockPassword,
  onProfileResent,
}: IOSSettingsModalProps) => {
  // the caller should ensure that hostMDMData is not undefined and that platform is supported otherwise we will allow an empty modal will be rendered.
  // https://fleetdm.com/handbook/company/why-this-way#why-make-it-obvious-when-stuff-breaks

  const memoizedTableData = useMemo(
    () => generateTableData(hostMDMData, platform),
    [hostMDMData, platform]
  );

  return (
    <Modal
      title="OS settings"
      onExit={onClose}
      className={baseClass}
      width="xlarge"
    >
      <OSSettingsTable
        canResendProfiles={canResendProfiles}
        canRotateRecoveryLockPassword={canRotateRecoveryLockPassword}
        tableData={memoizedTableData ?? []}
        resendRequest={resendRequest}
        rotateRecoveryLockPassword={rotateRecoveryLockPassword}
        onProfileResent={onProfileResent}
      />
      <div className="modal-cta-wrap">
        <Button onClick={onClose}>Done</Button>
      </div>
    </Modal>
  );
};

export default OSSettingsModal;
