import React from "react";
import { IUser } from "interfaces/user";
import Modal from "components/modals/Modal";
import Button from "components/buttons/Button";

const baseClass = "reset-password-modal";

interface IResetPasswordModal {
  user: IUser;
  modalBaseClass: string;
  onResetConfirm: (user: IUser) => void;
  onResetCancel: () => void;
}

const ResetPasswordModal = ({
  user,
  modalBaseClass,
  onResetConfirm,
  onResetCancel,
}: IResetPasswordModal): JSX.Element => {
  return (
    <Modal
      title="Require password reset"
      onExit={onResetCancel}
      className={`${modalBaseClass}__${baseClass}`}
    >
      <div className={baseClass}>
        <p>
          This user will be asked to reset their password after their next
          successful log in to Fleet.
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

export default ResetPasswordModal;
