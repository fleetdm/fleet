import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "delete-entra-tenant-modal";

interface IDeleteEntraTenantModalProps {
  tenantId: string;
  onExit: () => void;
}

const DeleteEntraTenantModal = ({
  tenantId,
  onExit,
}: IDeleteEntraTenantModalProps) => {
  const onDeleteToken = () => {
    onExit();
  };

  return (
    <Modal
      className={baseClass}
      title="Delete tenant"
      onExit={onExit}
      width="medium"
    >
      <>
        <p>
          This will stop both automatic (Autopilot) and manual enrollment by end
          users (<b>Settings &gt; Accounts &gt; Access work or school</b> on
          Windows) from this tenant.
        </p>
        <div className="modal-cta-wrap">
          <Button onClick={onExit} variant="alert">
            Delete
          </Button>
          <Button onClick={onExit} variant="inverse">
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default DeleteEntraTenantModal;
