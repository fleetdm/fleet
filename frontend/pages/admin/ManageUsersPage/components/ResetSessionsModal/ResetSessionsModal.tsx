import React from "react";
import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "reset-sessions-modal";

interface IResetSessionsModal {
  onResetConfirm: () => void;
  onResetCancel: () => void;
}

const ResetSessionsModal = ({
  onResetConfirm,
  onResetCancel,
}: IResetSessionsModal): JSX.Element => {
  return (
    <Modal
      title="Reset sessions"
      onExit={onResetCancel}
      onEnter={onResetConfirm}
    >
      <div className={baseClass}>
        <p>
          This user will be logged out of Fleet.
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

export default ResetSessionsModal;
