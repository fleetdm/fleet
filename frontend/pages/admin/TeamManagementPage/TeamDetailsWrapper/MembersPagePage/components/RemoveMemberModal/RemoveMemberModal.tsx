import React from "react";

import Modal from "components/modals/Modal";
import Button from "components/buttons/Button";

const baseClass = "remove-member-modal";

interface IDeleteTeamModalProps {
  memberName: string;
  teamName: string;
  onSubmit: () => void;
  onCancel: () => void;
}

const RemoveMemberModal = (props: IDeleteTeamModalProps): JSX.Element => {
  const { memberName, teamName, onSubmit, onCancel } = props;

  return (
    <Modal title={"Delete team"} onExit={onCancel} className={baseClass}>
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
    </Modal>
  );
};

export default RemoveMemberModal;
