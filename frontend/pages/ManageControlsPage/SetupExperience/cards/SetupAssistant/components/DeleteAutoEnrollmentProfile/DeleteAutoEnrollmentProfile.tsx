import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

interface DeleteAutoEnrollProfileProps {
  onCancel: () => void;
  onDelete: () => void;
}

const baseClass = "delete-auto-enrollment-profile-modal";

const DeleteAutoEnrollProfile = ({
  onCancel,
  onDelete,
}: DeleteAutoEnrollProfileProps) => {
  return (
    <Modal
      className={baseClass}
      title="Delete automatic enrollment profile"
      onExit={onCancel}
      onEnter={() => onDelete()}
    >
      <>
        <p>Delete the automatic enrollment profile to upload a new one.</p>
        <p>
          Without an automatic enrollment profile, new macOS hosts will
          automatically enroll with the default setup settings.
        </p>
        <div className="modal-cta-wrap">
          <Button type="button" onClick={() => onDelete()} variant="alert">
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
