import React, { useContext } from "react";

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

  const onDelete = async () => {
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

    onDeleted();
  };

  return (
    <Modal className={baseClass} title="Delete setup script" onExit={onExit}>
      <>
        <p>
          The script <b>{scriptName}</b> will still run on pending hosts.
        </p>
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
