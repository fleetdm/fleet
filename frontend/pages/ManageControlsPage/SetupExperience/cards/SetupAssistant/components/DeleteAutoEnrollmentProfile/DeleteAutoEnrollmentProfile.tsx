import React, { useContext } from "react";

import mdmAPI from "services/entities/mdm";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import { NotificationContext } from "context/notification";

interface DeleteAutoEnrollProfileProps {
  currentTeamId: number;
  onCancel: () => void;
  onDelete: () => void;
}

const baseClass = "delete-auto-enrollment-profile-modal";

const DeleteAutoEnrollProfile = ({
  currentTeamId,
  onCancel,
  onDelete,
}: DeleteAutoEnrollProfileProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const handleDelete = async () => {
    try {
      await mdmAPI.deleteSetupEnrollmentProfile(currentTeamId);
      renderFlash("success", "Successfully deleted!");
    } catch {
      renderFlash("error", "Couldn’t delete. Please try again.");
    }
    onDelete();
  };

  return (
    <Modal
      className={baseClass}
      title="Delete automatic enrollment profile"
      onExit={onCancel}
    >
      <>
        <p>Delete the automatic enrollment profile to upload a new one.</p>
        <p>
          Without an automatic enrollment profile, new macOS hosts will
          automatically enroll with the default setup settings.
        </p>
        <div className="modal-cta-wrap">
          <Button type="button" onClick={handleDelete} variant="alert">
            Delete
          </Button>
          <Button onClick={onCancel} variant="inverse-alert">
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default DeleteAutoEnrollProfile;
