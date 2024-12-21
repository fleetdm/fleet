import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "delete-label-modal";

interface IDeleteLabelModalProps {
  onSubmit: () => void;
  onCancel: () => void;
  isUpdatingLabel: boolean;
}

const DeleteLabelModal = ({
  onSubmit,
  onCancel,
  isUpdatingLabel,
}: IDeleteLabelModalProps): JSX.Element => {
  return (
    <Modal
      title="Delete label"
      onExit={onCancel}
      onEnter={onSubmit}
      className={baseClass}
    >
      <>
        <p>
          If a configuration profile uses this label as a custom target, the
          profile will break: it won&apos;t be applied to new hosts.
        </p>
        <p>
          To apply the profile to new hosts, you&apos;ll have to delete it and
          upload a new profile.
        </p>
        <div className="modal-cta-wrap">
          <Button
            onClick={onSubmit}
            variant="alert"
            className="delete-loading"
            isLoading={isUpdatingLabel}
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

export default DeleteLabelModal;
