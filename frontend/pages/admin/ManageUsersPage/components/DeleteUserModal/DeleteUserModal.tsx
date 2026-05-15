import React from "react";
import Button from "components/buttons/Button";
import Modal from "components/Modal";

const baseClass = "delete-user-form";

interface IDeleteUserModal {
  name: string;
  onDelete: () => void;
  onCancel: () => void;
  isUpdatingUsers: boolean;
}

const DeleteUserModal = ({
  name,
  onDelete,
  onCancel,
  isUpdatingUsers,
}: IDeleteUserModal): JSX.Element => {
  return (
    <Modal title="Delete user" onExit={onCancel} onEnter={onDelete}>
      <div className={baseClass}>
        <p>
          You are about to delete{" "}
          <span className={`${baseClass}__name`}>{name}</span> from Fleet.
        </p>
        <p className={`${baseClass}__warning`}>This action cannot be undone.</p>
        <div className="modal-cta-wrap">
          <Button
            type="button"
            variant="alert"
            onClick={onDelete}
            className="delete-loading"
            isLoading={isUpdatingUsers}
          >
            Delete
          </Button>
          <Button onClick={onCancel} variant="inverse-alert">
            Cancel
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default DeleteUserModal;
