import React from "react";
import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "reset-password-modal";

interface IResetPasswordModal {
  onResetConfirm: () => void;
  onResetCancel: () => void;
}

const ResetPasswordModal = ({
  onResetConfirm,
  onResetCancel,
}: IResetPasswordModal): JSX.Element => {
  return (
    <Modal
      title="Require password reset"
      onExit={onResetCancel}
      onEnter={onResetConfirm}
    >
      <div className={baseClass}>
        <p>
          This user will be asked to reset their password after their next
          successful log in to Fleet.
          <br />
          This will revoke all active Fleet API tokens for this user.
        </p>
        <div className="modal-cta-wrap">
          <Button type="button" onClick={onResetConfirm}>
            Confirm
          </Button>
          <Button onClick={onResetCancel} variant="inverse">
            Cancel
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default ResetPasswordModal;
