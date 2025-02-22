import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "remove-user-modal";

interface IDeleteTeamModalProps {
  userName: string;
  teamName: string;
  isUpdatingUsers: boolean;
  onSubmit: () => void;
  onCancel: () => void;
}

const RemoveUserModal = ({
  userName,
  teamName,
  isUpdatingUsers,
  onSubmit,
  onCancel,
}: IDeleteTeamModalProps): JSX.Element => {
  return (
    <Modal
      title="Remove user"
      onExit={onCancel}
      onEnter={onSubmit}
      className={baseClass}
    >
      <>
        <p>
          You are about to remove{" "}
          <span className={`${baseClass}__name`}>{userName}</span> from{" "}
          <span className={`${baseClass}__team-name`}>{teamName}</span>.
        </p>
        <p>
          If {userName} is not assigned to any other team, they will lose access
          to Fleet.
        </p>
        <div className="modal-cta-wrap">
          <Button
            type="button"
            variant="alert"
            onClick={onSubmit}
            className="remove-loading"
            isLoading={isUpdatingUsers}
          >
            Remove
          </Button>
          <Button onClick={onCancel} variant="inverse-alert">
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default RemoveUserModal;
