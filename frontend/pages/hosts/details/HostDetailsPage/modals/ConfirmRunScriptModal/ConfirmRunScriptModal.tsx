import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "confirm-run-script-modal";

interface IConfirmRunScriptModal {
  onCancel: () => void;
  onClose: () => void;
  onConfirmRunScript: () => void;
  scriptName?: string;
  hostName: string;
  isRunningScript: boolean;
  isHidden: boolean;
}

const ConfirmRunScriptModal = ({
  onCancel,
  onClose,
  onConfirmRunScript,
  scriptName,
  hostName,
  isRunningScript,
  isHidden,
}: IConfirmRunScriptModal) => {
  return (
    <Modal
      title="Run script?"
      onExit={onClose}
      isLoading={isRunningScript}
      isHidden={isHidden}
    >
      <form className={`${baseClass}__form`}>
        <p>
          {scriptName ? <b>{scriptName}</b> : "The script"} will run on{" "}
          <b>{hostName}</b>.
        </p>
        <div className="modal-cta-wrap">
          <Button
            type="button"
            onClick={onConfirmRunScript}
            className="save-loading"
            isLoading={isRunningScript}
          >
            Run
          </Button>
          <Button onClick={onCancel} variant="inverse">
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default ConfirmRunScriptModal;
