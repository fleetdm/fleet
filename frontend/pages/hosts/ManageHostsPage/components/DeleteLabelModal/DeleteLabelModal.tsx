import React, { useContext } from "react";

import { AppContext } from "context/app";

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
  const { isPremiumTier } = useContext(AppContext);
  return (
    <Modal
      title="Delete label"
      onExit={onCancel}
      onEnter={onSubmit}
      className={baseClass}
    >
      <>
        <p>Are you sure you wish to delete this label?</p>
        {isPremiumTier && (
          <ul>
            <li>
              Configuration profiles that target this label will not be applied
              to new hosts.
            </li>
            <li>
              Queries and policies that target this label will continue to run,
              but may target different hosts.
            </li>
          </ul>
        )}
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
