import React, { useContext } from "react";

import scriptAPI from "services/entities/scripts";
import { NotificationContext } from "context/notification";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "delete-script-modal";

interface IDeleteScriptModalProps {
  scriptName: string;
  scriptId: number;
  onCancel: () => void;
  onDone: () => void;
}

const DeleteScriptModal = ({
  scriptName,
  scriptId,
  onCancel,
  onDone,
}: IDeleteScriptModalProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const onClickDelete = async (id: number) => {
    try {
      await scriptAPI.deleteScript(id);
      renderFlash("success", "Successfully deleted!");
    } catch {
      renderFlash("error", "Couldn’t delete. Please try again.");
    }
    onDone();
  };

  return (
    <Modal
      className={baseClass}
      title="Delete script"
      onExit={onCancel}
      onEnter={() => onClickDelete(scriptId)}
    >
      <>
        <p>
          This action will cancel script{" "}
          <span className={`${baseClass}__script-name`}>{scriptName}</span> from
          running on macOS hosts on which the script hasn&apos;t run yet.
        </p>
        <div className="modal-cta-wrap">
          <Button
            type="button"
            onClick={() => onClickDelete(scriptId)}
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
