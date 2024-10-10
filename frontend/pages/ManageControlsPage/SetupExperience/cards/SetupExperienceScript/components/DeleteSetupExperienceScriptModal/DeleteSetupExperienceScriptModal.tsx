import Button from "components/buttons/Button";
import Modal from "components/Modal";
import React from "react";

const baseClass = "delete-setup-experience-script-modal";

interface IDeleteSetupExperienceScriptModalProps {
  onExit: () => void;
  onDeleted: () => void;
}

const DeleteSetupExperienceScriptModal = ({
  onExit,
  onDeleted,
}: IDeleteSetupExperienceScriptModalProps) => {
  const onDelete = () => {
    onDeleted();
  };

  return (
    <Modal className={baseClass} title="Delete setup script" onExit={onExit}>
      <>
        <p>Delete the setup script to upload a new one.</p>
        <div className="modal-cta-wrap">
          <Button type="button" onClick={onDelete} variant="alert">
            Delete
          </Button>
          <Button onClick={onExit} variant="inverse-alert">
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default DeleteSetupExperienceScriptModal;
