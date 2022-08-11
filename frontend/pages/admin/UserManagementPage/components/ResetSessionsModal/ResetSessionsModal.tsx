import React from "react";
import { IUser } from "interfaces/user";
import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "reset-sessions-modal";

interface IResetSessionsModal {
  user: IUser;
  onResetConfirm: (user: IUser) => void;
  onResetCancel: () => void;
}

const ResetSessionsModal = ({
  user,
  onResetConfirm,
  onResetCancel,
}: IResetSessionsModal): JSX.Element => {
  return (
    <Modal
      title="Reset sessions"
      onExit={onResetCancel}
      onEnter={() => onResetConfirm(user)}
    >
      <div className={baseClass}>
        <p>
          This user will be logged out of Fleet.
          <br />
          This will revoke all active Fleet API tokens for this user.
        </p>
        <div className={`${baseClass}__btn-wrap`}>
          <Button
            className={`${baseClass}__btn`}
            type="button"
            variant="brand"
            onClick={() => onResetConfirm(user)}
          >
            Confirm
          </Button>
          <Button
            className={`${baseClass}__btn`}
            onClick={onResetCancel}
            variant="inverse"
          >
            Cancel
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default ResetSessionsModal;
