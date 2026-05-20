import React from "react";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import { IPolicyStats } from "interfaces/policy";

const baseClass = "manage-automations-modal";

interface IManageAutomationsModalProps {
  policy: IPolicyStats;
  onExit: () => void;
}

// Placeholder – full implementation in follow-up PR
const ManageAutomationsModal = ({
  policy,
  onExit,
}: IManageAutomationsModalProps): JSX.Element => {
  return (
    <Modal title={policy.name} onExit={onExit} className={baseClass}>
      <div className="modal-cta-wrap">
        <Button onClick={onExit} variant="default">
          Done
        </Button>
      </div>
    </Modal>
  );
};

export default ManageAutomationsModal;
