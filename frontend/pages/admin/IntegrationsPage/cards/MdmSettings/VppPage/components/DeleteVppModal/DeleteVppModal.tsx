import React, { useCallback, useContext, useState } from "react";

import mdmAppleAPI from "services/entities/mdm_apple";
import { NotificationContext } from "context/notification";

import Button from "components/buttons/Button";
import Modal from "components/Modal";

const baseClass = "delete-vpp-modal";

interface IDeleteVppModalProps {
  orgName: string;
  tokenId: number;
  onCancel: () => void;
  onDeletedToken: () => void;
}

const DeleteVppModal = ({
  orgName,
  tokenId,
  onCancel,
  onDeletedToken,
}: IDeleteVppModalProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const [isDeleting, setIsDeleting] = useState(false);

  const onDeleteToken = useCallback(async () => {
    setIsDeleting(true);

    try {
      await mdmAppleAPI.deleteVppToken(tokenId);
      renderFlash("success", "Deleted successfully.");
      onDeletedToken();
    } catch (e) {
      // TODO: Check API sends back correct error messages
      renderFlash("error", "Couldn’t delete. Please try again.");
      onCancel();
    }
  }, [onCancel, onDeletedToken, renderFlash, tokenId]);

  return (
    <Modal
      title="Delete VPP"
      className={baseClass}
      onExit={onCancel}
      isContentDisabled={isDeleting}
    >
      <>
        <p>
          Apps purchased for the <b>{orgName}</b> location won&apos;t appear in
          Fleet, and app install policy automations will be removed. Apps
          won&apos;t be uninstalled from hosts.
        </p>
        <p>
          If you want to enable VPP integration again, you&apos;ll have to
          upload a new token.
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

export default DeleteVppModal;
