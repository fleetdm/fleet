import React, { useMemo } from "react";
import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { IHostMdmData } from "interfaces/host";

import OSSettingsTable from "./OSSettingsTable";
import { generateTableData } from "./OSSettingsTable/OSSettingsTableConfig";

interface IOSSettingsModalProps {
  hostId: number;
  platform: string;
  hostMDMData: IHostMdmData;
  /** controls showing the action for a user to resend a profile. Defaults to `false` */
  canResendProfiles?: boolean;
  onClose: () => void;
  /** handler that fires when a profile was reset. Requires `canResendProfiles` prop
   * to be `true`, otherwise has no effect.
   */
  onProfileResent?: () => void;
}

const baseClass = "os-settings-modal";

const OSSettingsModal = ({
  hostId,
  platform,
  hostMDMData,
  canResendProfiles = false,
  onClose,
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
      <>
        <OSSettingsTable
          canResendProfiles={canResendProfiles}
          hostId={hostId}
          tableData={memoizedTableData ?? []}
          onProfileResent={onProfileResent}
        />
        <div className="modal-cta-wrap">
          <Button variant="brand" onClick={onClose}>
            Done
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default OSSettingsModal;
