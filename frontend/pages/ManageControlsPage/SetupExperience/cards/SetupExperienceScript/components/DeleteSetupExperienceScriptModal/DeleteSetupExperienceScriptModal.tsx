import React, { useContext, useState } from "react";

import mdmAPI from "services/entities/mdm";
import { NotificationContext } from "context/notification";

import Button from "components/buttons/Button";
import Modal from "components/Modal";

const baseClass = "delete-setup-experience-script-modal";

interface IDeleteSetupExperienceScriptModalProps {
  currentTeamId: number;
  scriptName: string;
  onExit: () => void;
  onDeleted: () => void;
}

const DeleteSetupExperienceScriptModal = ({
  currentTeamId,
  scriptName,
  onExit,
  onDeleted,
}: IDeleteSetupExperienceScriptModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isDeleting, setIsDeleting] = useState(false);

  const onDelete = async () => {
    setIsDeleting(true);
    try {
      await mdmAPI.deleteSetupExperienceScript(currentTeamId);
      renderFlash("success", "Setup script successfully deleted!");
    } catch (error) {
      renderFlash(
        "error",
        "Couldn't delete the setup script. Please try again."
      );
      console.error(error);
    }
    setIsDeleting(false);
    onDeleted();
  };

  return (
    <Modal
      className={baseClass}
      title="Delete setup script"
      onExit={onExit}
      isContentDisabled={isDeleting}
    >
      <>
        <p>
          This action will cancel any pending script execution for{" "}
          <b>{scriptName}</b>.
        </p>
        <p>
          If the script is currently running on a host it will still complete,
          but results won&apos;t appear in Fleet.
        </p>
        <p>You cannot undo this action.</p>
        <div className="modal-cta-wrap">
          <Button
            type="button"
            onClick={onDelete}
            variant="alert"
            isLoading={isDeleting}
          >
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
