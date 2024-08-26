import React, { useCallback, useContext, useState } from "react";

import mdmAbmAPI from "services/entities/mdm_apple_bm";
import { IMdmAbmToken } from "interfaces/mdm";
import { NotificationContext } from "context/notification";

import Button from "components/buttons/Button";
import Modal from "components/Modal";

const baseClass = "delete-abm-modal";

interface IDeleteAbmModalProps {
  tokenOrgName: string;
  tokenId: number;
  onCancel: () => void;
  onDeletedToken: () => void;
}

const DeleteAbmModal = ({
  tokenOrgName,
  tokenId,
  onCancel,
  onDeletedToken,
}: IDeleteAbmModalProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const [isDeleting, setIsDeleting] = useState(false);

  const onDeleteToken = useCallback(async () => {
    setIsDeleting(true);

    try {
      await mdmAbmAPI.deleteToken(tokenId);
      renderFlash("success", "Deleted successfully.");
      onDeletedToken();
    } catch (e) {
      // TODO: Check API sends back correct error messages
      renderFlash(
        "error",
        "Couldn’t disable automatic enrollment. Please try again."
      );
      onCancel();
    }
  }, [onCancel, onDeletedToken, renderFlash, tokenId]);

  return (
    <Modal
      title="Delete ABM"
      className={baseClass}
      onExit={onCancel}
      isContentDisabled={isDeleting}
    >
      <>
        <p>
          New hosts purchased in the <b>{tokenOrgName}</b> won&apos;t
          automatically enroll to Fleet.{" "}
        </p>
        <p>
          If you want to re-enable automatic enrollment, you&apos;ll have to
          upload a new ABM token.
        </p>

        <div className="modal-cta-wrap">
          <Button
            type="button"
            variant="alert"
            onClick={onDeleteToken}
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

export default DeleteAbmModal;
