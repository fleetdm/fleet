import React, { useEffect } from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "remove-member-modal";

interface IDeleteTeamModalProps {
  memberName: string;
  teamName: string;
  onSubmit: () => void;
  onCancel: () => void;
}

const RemoveMemberModal = ({
  memberName,
  teamName,
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
    <Modal title={"Remove team member"} onExit={onCancel} className={baseClass}>
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
