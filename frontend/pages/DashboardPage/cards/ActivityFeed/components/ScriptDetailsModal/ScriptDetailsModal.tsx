import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import FleetMarkdown from "components/FleetMarkdown/FleetMarkdown";

const baseClass = "scripts-details-modal";

interface IScriptDetailsModalProps {
  onCancel: () => void;
}

const ScriptDetailsModal = ({ onCancel }: IScriptDetailsModalProps) => {
  return (
    <Modal
      title={"Script Details"}
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
    >
      <div>
        <span>Script content:</span>
        <FleetMarkdown markdown="```test here```" />
      </div>
      <div className="modal-cta-wrap">
        <Button onClick={onCancel} variant="brand">
          Done
        </Button>
      </div>
    </Modal>
  );
};

export default ScriptDetailsModal;
