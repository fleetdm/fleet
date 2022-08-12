import React, { useEffect } from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";

const baseClass = "remove-member-modal";

interface IDeleteTeamModalProps {
  memberName: string;
  teamName: string;
  isLoading: boolean;
  onSubmit: () => void;
  onCancel: () => void;
}

const RemoveMemberModal = ({
  memberName,
  teamName,
  isLoading,
  onSubmit,
  onCancel,
}: IDeleteTeamModalProps): JSX.Element => {
  return (
    <Modal
      title={"Remove team member"}
      onExit={onCancel}
      onEnter={onSubmit}
      className={baseClass}
    >
      {isLoading ? (
        <Spinner />
      ) : (
        <form className={`${baseClass}__form`}>
          <p>
            You are about to remove{" "}
            <span className={`${baseClass}__name`}>{memberName}</span> from{" "}
            <span className={`${baseClass}__team-name`}>{teamName}</span>.
          </p>
          <p>
            If {memberName} is not a member of any other team, they will lose
            access to Fleet.
          </p>
          <div className={`${baseClass}__btn-wrap`}>
            <Button
              className={`${baseClass}__btn`}
              type="button"
              variant="alert"
              onClick={onSubmit}
            >
              Remove
            </Button>
            <Button
              className={`${baseClass}__btn`}
              onClick={onCancel}
              variant="inverse-alert"
            >
              Cancel
            </Button>
          </div>
        </form>
      )}
    </Modal>
  );
};

export default RemoveMemberModal;
