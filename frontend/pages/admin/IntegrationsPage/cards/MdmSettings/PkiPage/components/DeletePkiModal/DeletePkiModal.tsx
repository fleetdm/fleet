import React, { useCallback, useContext, useState } from "react";

import { IPkiConfig } from "interfaces/pki";
import pkiAPI from "services/entities/pki";

import { NotificationContext } from "context/notification";

import Button from "components/buttons/Button";
import Modal from "components/Modal";

const baseClass = "delete-pki-modal";

interface IDeletePkiModalProps {
  pkiConfig: IPkiConfig;
  onCancel: () => void;
  onDeleted: () => void;
}

const DeletePkiModal = ({
  pkiConfig: { pki_name: name },
  onCancel,
  onDeleted,
}: IDeletePkiModalProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const [isDeleting, setIsDeleting] = useState(false);

  const onDelete = useCallback(async () => {
    setIsDeleting(true);

    try {
      await pkiAPI.deleteCert(name);
      renderFlash("success", "Deleted successfully.");
      onDeleted();
    } catch (e) {
      // TODO: Check API sends back correct error messages
      renderFlash("error", "Couldn’t delete. Please try again.");
      onCancel();
    }
  }, [onCancel, onDeleted, renderFlash, name]);

  return (
    <Modal
      title="Delete PKI"
      className={baseClass}
      onExit={onCancel}
      isContentDisabled={isDeleting}
    >
      <>
        <p>
          If you want to re-enable PKI, you&apos;ll have to upload a new RA
          certificate.
        </p>

        <div className="modal-cta-wrap">
          <Button
            type="button"
            variant="alert"
            onClick={onDelete}
            disabled={isDeleting}
            isLoading={isDeleting}
          >
            Delete
          </Button>
          <Button
            onClick={onCancel}
            disabled={isDeleting}
            variant="inverse-alert"
          >
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default DeletePkiModal;
