import React, { useCallback, useContext } from "react";

import softwareAPI from "services/entities/software";
import { NotificationContext } from "context/notification";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "delete-software-modal";

interface IDeleteSoftwareModalProps {
  softwareId: number;
  teamId: number;
  onExit: () => void;
  onSuccess: () => void;
}

const DeleteSoftwareModal = ({
  softwareId,
  teamId,
  onExit,
  onSuccess,
}: IDeleteSoftwareModalProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const onDeleteSoftware = useCallback(async () => {
    try {
      await softwareAPI.deleteSoftwarePackage(softwareId, teamId);
      renderFlash("success", "Software deleted successfully!");
      onSuccess();
    } catch {
      renderFlash("error", "Couldn't delete. Please try again.");
    }
    onExit();
  }, [softwareId, teamId, renderFlash, onSuccess, onExit]);

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
