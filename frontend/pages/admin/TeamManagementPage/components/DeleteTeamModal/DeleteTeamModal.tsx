import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "delete-team-modal";

interface IDeleteTeamModalProps {
  name: string;
  isUpdatingTeams: boolean;
  onSubmit: () => void;
  onCancel: () => void;
}

const DeleteTeamModal = ({
  name,
  isUpdatingTeams,
  onSubmit,
  onCancel,
}: IDeleteTeamModalProps): JSX.Element => {
  return (
    <Modal
      title="Delete team"
      onExit={onCancel}
      onEnter={onSubmit}
      className={baseClass}
    >
      <>
        <p>
          You are about to delete{" "}
          <span className={`${baseClass}__name`}>{name}</span> from Fleet.
        </p>
        <p>
          Users on this team who are not assigned to other teams will lose
          access to Fleet.
        </p>
        <p className={`${baseClass}__warning`}>This action cannot be undone.</p>
        <div className="modal-cta-wrap">
          <Button
            type="button"
            onClick={onSubmit}
            variant="alert"
            className="delete-loading"
            isLoading={isUpdatingTeams}
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

export default DeleteTeamModal;
