import React from "react";

import Button from "components/buttons/Button";
import Modal from "components/Modal";

import { EULA_PLATFORM_CONFIG, EulaPlatform } from "../../helpers";

const baseClass = "delete-eula-modal";

interface IDeleteEulaModalProps {
  platform: EulaPlatform;
  isDeleting: boolean;
  onDelete: () => void;
  onCancel: () => void;
}

const DeleteEulaModal = ({
  platform,
  isDeleting,
  onDelete,
  onCancel,
}: IDeleteEulaModalProps) => {
  return (
    <Modal
      className={baseClass}
      title="Delete EULA"
      onExit={onCancel}
      onEnter={onDelete}
      isContentDisabled={isDeleting}
    >
      <>
        <p>{EULA_PLATFORM_CONFIG[platform].deleteMessage}</p>
        <div className="modal-cta-wrap">
          <Button
            type="button"
            onClick={onDelete}
            variant="alert"
            isLoading={isDeleting}
          >
            Delete
          </Button>
          <Button onClick={onCancel} variant="secondary">
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default DeleteEulaModal;
