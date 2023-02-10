import React from "react";
import Button from "components/buttons/Button";
import Modal from "components/Modal";

interface IMacSettingsModalProps {
  hostId: number;
  hostMacSettings: unknown; // TODO: define this type when API shape is determined
  onClose: () => void;
}

const baseClass = "mac-settings-modal";

const MacSettingsModal = ({
  hostId,
  hostMacSettings,
  onClose,
}: IMacSettingsModalProps) => {
  return (
    <Modal title="macOS settings" onExit={onClose} className={baseClass}>
      <div className="modal-cta-wrap">
        <Button type="submit" variant="brand" onClick={onClose}>
          Done
        </Button>
      </div>
    </Modal>
  );
};

export default MacSettingsModal;
