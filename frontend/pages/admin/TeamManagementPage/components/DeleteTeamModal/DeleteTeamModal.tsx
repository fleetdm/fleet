import React, { useEffect } from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";

const baseClass = "delete-team-modal";

interface IDeleteTeamModalProps {
  name: string;
  teamIsRemoving: boolean;
  onSubmit: () => void;
  onCancel: () => void;
}

const DeleteTeamModal = ({
  name,
  teamIsRemoving,
  onSubmit,
  onCancel,
}: IDeleteTeamModalProps): JSX.Element => {
  useEffect(() => {
    const listener = (event: KeyboardEvent) => {
      if (event.code === "Enter" || event.code === "NumpadEnter") {
        event.preventDefault();
        onSubmit();
      }
    };
    document.addEventListener("keydown", listener);
    return () => {
      document.removeEventListener("keydown", listener);
    };
  }, []);

  return (
    <Modal title={"Delete team"} onExit={onCancel} className={baseClass}>
      {teamIsRemoving ? (
        <Spinner />
      ) : (
        <form className={`${baseClass}__form`}>
          <p>
            You are about to delete{" "}
            <span className={`${baseClass}__name`}>{name}</span> from Fleet.
          </p>
          <p>
            Members of this team who are not members of other teams will lose
            access to Fleet.
          </p>
          <p className={`${baseClass}__warning`}>
            This action cannot be undone.
          </p>
          <div className="modal-cta-wrap">
            <Button type="button" onClick={onSubmit} variant="alert">
              Delete
            </Button>
            <Button onClick={onCancel} variant="inverse-alert">
              Cancel
            </Button>
          </div>
        </form>
      )}
    </Modal>
  );
};

export default DeleteTeamModal;
