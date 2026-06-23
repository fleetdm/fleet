import React, { useCallback, useState } from "react";

import mdmAbmAPI from "services/entities/mdm_apple_bm";

import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { notify } from "components/ToastNotification";

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
  const [isDeleting, setIsDeleting] = useState(false);

  const onDeleteToken = useCallback(async () => {
    setIsDeleting(true);

    try {
      await mdmAbmAPI.deleteToken(tokenId);
      notify.success("Deleted successfully.");
      onDeletedToken();
    } catch (e) {
      // TODO: Check API sends back correct error messages
      notify.error("Couldn’t disable automatic enrollment. Please try again.", {
        response: e,
      });
      onCancel();
    }
  }, [onCancel, onDeletedToken, tokenId]);

  return (
    <Modal
      title="Delete AB"
      className={baseClass}
      onExit={onCancel}
      isContentDisabled={isDeleting}
    >
      <p>
        New hosts purchased in the <b>{tokenOrgName}</b> won&apos;t
        automatically enroll to Fleet.{" "}
      </p>
      <p>
        If you want to re-enable automatic enrollment, you&apos;ll have to
        upload a new AB token.
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
    </Modal>
  );
};

export default DeleteAbmModal;
