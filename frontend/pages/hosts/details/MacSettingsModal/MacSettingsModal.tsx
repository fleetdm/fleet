import React, { useMemo } from "react";
import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { IHostMdmData } from "interfaces/host";

import MacSettingsTable from "./MacSettingsTable";
import { generateTableData } from "./MacSettingsTable/MacSettingsTableConfig";

interface IMacSettingsModalProps {
  hostMDMData?: Pick<IHostMdmData, "profiles" | "macos_settings">;
  onClose: () => void;
}

const baseClass = "mac-settings-modal";

const MacSettingsModal = ({ hostMDMData, onClose }: IMacSettingsModalProps) => {
  const memoizedTableData = useMemo(() => generateTableData(hostMDMData), [
    hostMDMData,
  ]);

  return (
    <Modal
      title="macOS settings"
      onExit={onClose}
      className={baseClass}
      width="large"
    >
      <>
        <MacSettingsTable tableData={memoizedTableData} />
        <div className="modal-cta-wrap">
          <Button variant="brand" onClick={onClose}>
            Done
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default MacSettingsModal;
