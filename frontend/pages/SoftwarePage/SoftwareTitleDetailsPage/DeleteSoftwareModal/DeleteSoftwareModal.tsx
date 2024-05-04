import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "delete-software-modal";

interface IDeleteSoftwareModalProps {
  softwareId: number;
  onExit: () => void;
}

const DeleteSoftwareModal = ({
  softwareId,
  onExit,
}: IDeleteSoftwareModalProps) => {
  const onDeleteSoftware = () => {
    console.log("Delete software");
    onExit();
  };

  return (
    <Modal className={baseClass} title="Delete software" onExit={onExit}>
      <>
        <p>Software won&apos;t be uninstalled from existing hosts.</p>
        <div className="modal-cta-wrap">
          <Button variant="alert" onClick={onDeleteSoftware}>
            Delete
          </Button>
          <Button variant="inverse-alert" onClick={onExit}>
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default DeleteSoftwareModal;
