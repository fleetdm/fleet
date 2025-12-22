import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "cancel-script-batch-modal";

interface ICancelScriptBatchModal {
  onExit: () => void;
  onSubmit: () => void;
  scriptName?: string;
  isCanceling: boolean;
}

const CancelScriptBatchModal = ({
  onSubmit,
  onExit,
  scriptName,
  isCanceling,
}: ICancelScriptBatchModal) => {
  return (
    <Modal
      title="Cancel script?"
      onExit={onExit}
      onEnter={onSubmit}
      className={baseClass}
    >
      <>
        <div className={`${baseClass}__content`}>
          <p>
            This will cancel any pending script runs for{" "}
            {scriptName ? <b>{scriptName}</b> : "this batch"}.
          </p>
          <p>
            If this script is currently running on a host, it will complete, but
            results won&rsquo;t appear in Fleet.
          </p>
          <p>You cannot undo this action.</p>
          <div className="modal-cta-wrap">
            <Button
              isLoading={isCanceling}
              disabled={isCanceling}
              onClick={onSubmit}
              variant="alert"
            >
              Cancel script
            </Button>
            <Button variant="inverse-alert" onClick={onExit}>
              Back
            </Button>
          </div>
        </div>
      </>
    </Modal>
  );
};

export default CancelScriptBatchModal;
