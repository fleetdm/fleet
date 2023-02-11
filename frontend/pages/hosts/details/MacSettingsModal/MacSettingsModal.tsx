import React from "react";
import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { IMacSettings } from "interfaces/mdm";
import MacSettingsTable from "./MacSettingsTable";

interface IMacSettingsModalProps {
  hostMacSettings?: IMacSettings; // TODO: define this type when API shape is determined
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
        <MacSettingsTable isLoading={false} hostMacSettings={hostMacSettings} />
        <div className="modal-cta-wrap">
          <Button type="submit" variant="brand" onClick={onClose}>
            Done
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default MacSettingsModal;
