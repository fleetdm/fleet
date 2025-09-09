import React, { useContext, useState } from "react";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import { ISecret } from "interfaces/secrets";
import { NotificationContext } from "context/notification";

import formatErrorResponse from "utilities/format_error_response";
import secretsAPI from "services/entities/secrets";

interface DeleteSecretModalProps {
  secret: ISecret | undefined;
  onExit: () => void;
  reloadList: () => void;
}

const baseClass = "fleet-delete-secret-modal";

const DeleteSecretModal = ({
  secret,
  onExit,
  reloadList,
}: DeleteSecretModalProps) => {
  const [isDeleting, setIsDeleting] = useState(false);

  const { renderFlash } = useContext(NotificationContext);

  const onClickDelete = async () => {
    if (!secret) {
      return;
    }
    setIsDeleting(true);
    try {
      await secretsAPI.deleteSecret(secret.id);
      renderFlash("success", "Variable successfully deleted.");
      reloadList();
    } catch (error) {
      const errorObject = formatErrorResponse(error);
      const isInUseError =
        errorObject.http_status === 409 &&
        /used by/.test(errorObject?.base ?? "");
      const message =
        isInUseError && typeof errorObject?.base === "string"
          ? errorObject.base
          : "An error occurred while deleting the custom variable. Please try again.";
      renderFlash("error", message);
    } finally {
      setIsDeleting(false);
      onExit();
    }
  };

  return (
    <Modal
      title="Delete custom variable?"
      onExit={onExit}
      className={baseClass}
    >
      <>
        <div className={`${baseClass}__message`}>
          <span>
            This will delete the
            <b>
              <TooltipTruncatedText value={secret?.name} />
            </b>
            custom variable.
          </span>
          <br />
          <br />
          If this custom variable is used in any configuration profiles or
          scripts, they will fail. To resolve, edit the configuration profile or
          script.
        </div>
        <div className="modal-cta-wrap">
          <Button
            variant="alert"
            onClick={onClickDelete}
            isLoading={isDeleting}
            disabled={isDeleting}
          >
            Delete
          </Button>
          <Button variant="inverse-alert" onClick={onExit}>
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default DeleteSecretModal;
