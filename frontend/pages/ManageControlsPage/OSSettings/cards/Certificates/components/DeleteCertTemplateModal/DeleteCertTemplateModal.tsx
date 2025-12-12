import React, { useContext, useState } from "react";

import certAPI, { ICertTemplate } from "services/entities/certificates";
import { NotificationContext } from "context/notification";

import Button from "components/buttons/Button";
import Modal from "components/Modal";

const baseClass = "delete-cert-template-modal";

interface IDeleteCertTemplateModalProps {
  cT: ICertTemplate;
  onExit: () => void;
}

const DeleteCertTemplateModal = ({
  cT,
  onExit,
}: IDeleteCertTemplateModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isUpdating, setIsUpdating] = useState(false);

  const { name, id } = cT;

  const onDelete = async () => {
    setIsUpdating(true);
    try {
      await certAPI.deleteCertTemplate(id);
      renderFlash("success", "Successfully deleted certificate template.");
      setIsUpdating(false);
      onExit();
    } catch (e) {
      setIsUpdating(false);
      renderFlash(
        "error",
        "Couldn't delete certificate template. Please try again."
      );
    }
  };

  return (
    <Modal
      className={baseClass}
      title="Delete certificate template"
      onExit={onExit}
    >
      <>
        <p>
          This action will remove the <b>{name}</b> certificate from all hosts
          assigned to this team.
        </p>
        <div className="modal-cta-wrap">
          <Button
            variant="alert"
            onClick={onDelete}
            isLoading={isUpdating}
            disabled={isUpdating}
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

export default DeleteCertTemplateModal;
