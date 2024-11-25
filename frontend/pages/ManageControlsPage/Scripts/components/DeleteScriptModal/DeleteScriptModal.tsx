import React, { useContext } from "react";

import scriptAPI from "services/entities/scripts";
import { NotificationContext } from "context/notification";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import { AxiosResponse } from "axios";
import { IApiError } from "../../../../../interfaces/errors";
import { getErrorMessage } from "../ScriptUploader/helpers";

const baseClass = "delete-script-modal";

interface IDeleteScriptModalProps {
  scriptName: string;
  scriptId: number;
  onCancel: () => void;
  onDone: () => void;
  isHidden?: boolean;
}

const DeleteScriptModal = ({
  scriptName,
  scriptId,
  onCancel,
  onDone,
  isHidden = false,
}: IDeleteScriptModalProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const onClickDelete = async (id: number) => {
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
    onDone();
  };

  return (
    <Modal
      className={baseClass}
      title="Delete script"
      onExit={onCancel}
      onEnter={() => onClickDelete(scriptId)}
      isHidden={isHidden}
    >
      <>
        <p>
          The script{" "}
          <span className={`${baseClass}__script-name`}>{scriptName}</span> will
          run on pending hosts. After the script runs, its output and exit code
          will appear in the activity feed.
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
