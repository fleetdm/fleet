import React from "react";
import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "automations-modal";

interface IAutomationsModalProps {
  onExit: () => void;
}

// Placeholder – full implementation in follow-up PR
const AutomationsModal = ({ onExit }: IAutomationsModalProps): JSX.Element => {
  return (
    <Modal title="Automations" onExit={onExit} className={baseClass}>
      <div className="modal-cta-wrap">
        <Button onClick={onExit} variant="default">
          Done
        </Button>
      </div>
    </Modal>
  );
};

export default AutomationsModal;
