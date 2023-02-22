import React from "react";
import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { IMacSettings } from "interfaces/mdm";
import MacSettingsTable from "./MacSettingsTable";

interface IMacSettingsModalProps {
  hostMacSettings?: IMacSettings;
  onClose: () => void;
}

const baseClass = "mac-settings-modal";

const MacSettingsModal = ({
  hostMacSettings,
  onClose,
}: IMacSettingsModalProps) => {
  return (
    <Modal title="macOS settings" onExit={onClose} className={baseClass}>
      <>
        <MacSettingsTable hostMacSettings={hostMacSettings} />
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
