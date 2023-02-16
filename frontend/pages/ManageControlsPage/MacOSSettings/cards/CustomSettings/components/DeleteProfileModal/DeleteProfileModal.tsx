import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

interface DeleteProfileModalProps {
  profileName: string;
  profileId: number;
  onCancel: () => void;
  onDelete: (profileId: number) => void;
}

const baseClass = "delete-profile-modal";

const DeleteProfileModal = ({
  profileName,
  profileId,
  onCancel,
  onDelete,
}: DeleteProfileModalProps) => {
  return (
    <Modal
      className={baseClass}
      title={"Delete configuration profile"}
      onExit={onCancel}
      onEnter={() => onDelete(profileId)}
    >
      <>
        <p>
          This action will delete configuration profile{" "}
          <span>{profileName}</span> from all macOS hosts assigned to this team.
        </p>
        <div className="modal-cta-wrap">
          <Button
            type="button"
            onClick={() => onDelete(profileId)}
            variant="alert"
            className="delete-loading"
          >
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

export default DeleteProfileModal;
