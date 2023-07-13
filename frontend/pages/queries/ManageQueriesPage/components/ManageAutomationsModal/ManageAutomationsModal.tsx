import React from "react";

import Modal from "components/Modal";

const baseClass = "automations-modal";

interface IManageAutomationsModalProps {
  onExit: () => void;
}

const ManageAutomationsModal = ({
  onExit,
}: IManageAutomationsModalProps): JSX.Element => {
  return (
    <Modal title={"Manage automations"} onExit={onExit} className={baseClass}>
      <div className={baseClass} />
    </Modal>
  );
};

export default ManageAutomationsModal;
