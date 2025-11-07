import React, { useContext, useState } from "react";

import scriptAPI from "services/entities/scripts";
import { NotificationContext } from "context/notification";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import { AxiosResponse } from "axios";
import { IApiError } from "../../../../../interfaces/errors";
import { getErrorMessage } from "../ScriptUploadModal/helpers";

const baseClass = "delete-script-modal";

interface IDeleteScriptModalProps {
  scriptName: string;
  scriptId: number;
  onCancel: () => void;
  afterDelete: () => void;
  isHidden?: boolean;
}

const DeleteScriptModal = ({
  scriptName,
  scriptId,
  onCancel,
  afterDelete,
  isHidden = false,
}: IDeleteScriptModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isDeleting, setIsDeleting] = useState(false);

  const onClickDelete = async (id: number) => {
    setIsDeleting(true);
    try {
      await scriptAPI.deleteScript(id);
      renderFlash("success", "Successfully deleted!");
    } catch (e) {
      const error = e as AxiosResponse<IApiError>;
      const apiErrMessage = getErrorMessage(error);
      renderFlash(
        "error",
        apiErrMessage.includes("Policy automation")
          ? apiErrMessage
          : "Couldn’t delete. Please try again."
      );
    }
    setIsDeleting(false);
    afterDelete();
  };

  return (
    <Modal
      className={baseClass}
      title="Delete script"
      onExit={onCancel}
      onEnter={() => onClickDelete(scriptId)}
      isHidden={isHidden}
      isContentDisabled={isDeleting}
    >
      <>
        <p>
          This action will cancel any pending script execution for{" "}
          <span className={`${baseClass}__script-name`}>{scriptName}</span>
        </p>
        <p>
          If the script is currently running on a host it will still complete,
          but results won&apos;t appear in Fleet.
        </p>
        <p>You cannot undo this action.</p>
        <div className="modal-cta-wrap">
          <Button
            type="button"
            onClick={() => onClickDelete(scriptId)}
            variant="alert"
            className="delete-loading"
            isLoading={isDeleting}
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
