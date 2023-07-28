import React from "react";

import Modal from "components/Modal";

const baseClass = "automations-modal";

interface IAutomationsModalProps {
  onExit: () => void;
}

const AutomationsModal = ({ onExit }: IAutomationsModalProps): JSX.Element => {
  return (
    <Modal title={"Manage automations"} onExit={onExit} className={baseClass}>
      <div className={baseClass} />
    </Modal>
  );
};

export default AutomationsModal;
