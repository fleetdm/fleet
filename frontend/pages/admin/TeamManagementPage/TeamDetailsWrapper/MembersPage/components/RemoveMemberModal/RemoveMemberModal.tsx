import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "remove-member-modal";

interface IDeleteTeamModalProps {
  memberName: string;
  teamName: string;
  isUpdatingMembers: boolean;
  onSubmit: () => void;
  onCancel: () => void;
}

const RemoveMemberModal = ({
  memberName,
  teamName,
  isUpdatingMembers,
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
      <>
        <p>
          You are about to remove{" "}
          <span className={`${baseClass}__name`}>{memberName}</span> from{" "}
          <span className={`${baseClass}__team-name`}>{teamName}</span>.
        </p>
        <p>
          If {memberName} is not a member of any other team, they will lose
          access to Fleet.
        </p>
        <div className="modal-cta-wrap">
          <Button
            type="button"
            variant="alert"
            onClick={onSubmit}
            className="remove-loading"
            isLoading={isUpdatingMembers}
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

export default RemoveMemberModal;
