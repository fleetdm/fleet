import React, { useCallback, useState } from "react";

import mdmAppleAPI from "services/entities/mdm_apple";

import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { notify } from "components/ToastNotification";

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
  const [isDeleting, setIsDeleting] = useState(false);

  const onDeleteToken = useCallback(async () => {
    setIsDeleting(true);

    try {
      await mdmAppleAPI.deleteVppToken(tokenId);
      notify.success("Deleted successfully.");
      onDeletedToken();
    } catch (e) {
      // TODO: Check API sends back correct error messages
      notify.error("Couldn’t delete. Please try again.", { response: e });
      onCancel();
    }
  }, [onCancel, onDeletedToken, tokenId]);

  return (
    <Modal
      title="Delete VPP"
      className={baseClass}
      onExit={onCancel}
      isContentDisabled={isDeleting}
    >
      <p>
        Apps purchased for the <b>{orgName}</b> organization unit won&apos;t
        appear in Fleet, and policies that trigger automatic install of these
        apps will be deleted. Apps won&apos;t be uninstalled from hosts.
      </p>
      <p>
        If you want to enable VPP integration again, you&apos;ll have to upload
        a new token.
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

export default DeleteVppModal;
