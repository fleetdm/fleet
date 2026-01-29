import Button from "components/buttons/Button";
import Modal from "components/Modal";
import React from "react";

const baseClass = "device-bypass-modal";

interface IBypassModal {
  onCancel: () => void;
  onResolveLater: () => void;
  isLoading: boolean;
}

const BypassModal = ({ onCancel, onResolveLater, isLoading }: IBypassModal) => {
  return (
    <Modal onExit={onCancel} title="Resolve later">
      <>
        <p>
          This will allow you to log in with Okta once.
          <br />
          <br />
         Please resolve all policies marked &quot;Action required&quot; to restore access for subsequent logins.
        </p>
        <div className="modal-cta-wrap">
          <Button
            type="button"
            onClick={onResolveLater}
            isLoading={isLoading}
            disabled={isLoading}
          >
            Resolve later
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default BypassModal;
