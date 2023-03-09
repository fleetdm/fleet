import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "delete-script-modal";

interface IDeleteScriptModalProps {
  scriptName: string;
  scriptId: number;
  onCancel: () => void;
  onDelete: (scriptId: number) => void;
}

const DeleteScriptModal = ({
  scriptName,
  scriptId,
  onCancel,
  onDelete,
}: IDeleteScriptModalProps) => {
  return (
    <Modal
      className={baseClass}
      title={"Delete script"}
      onExit={onCancel}
      onEnter={() => onDelete(scriptId)}
    >
      <>
        <p>
          This action will cancel script{" "}
          <span className={`${baseClass}__script-name`}>{scriptName}</span> from
          running on macOS hosts on which the scrupt hasn&apos;t run yet.
        </p>
        <div className="modal-cta-wrap">
          <Button
            type="button"
            onClick={() => onDelete(scriptId)}
            variant="alert"
            className="delete-loading"
          >
            Delete
          </Button>
          <Button onClick={onCancel} variant="inverse-alert">
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default DeleteScriptModal;
