import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "delete-asset-modal";

interface IDeleteAssetModalProps {
  assetUuid: string;
  onCancel: () => void;
  onDelete: (assetUuid: string) => void;
  isDeleting: boolean;
}

const DeleteAssetModal = ({
  assetUuid,
  onCancel,
  onDelete,
  isDeleting,
}: IDeleteAssetModalProps) => {
  return (
    <Modal
      className={baseClass}
      title="Delete asset"
      onExit={onCancel}
      onEnter={() => onDelete(assetUuid)}
      width="large"
    >
      <>
        <div className={`${baseClass}__content`}>
          <p>
            Assets that are linked in a configuration profile will not be
            deleted. You will need to delete the configuration profile first.
          </p>
        </div>
        <div className="modal-cta-wrap">
          <Button
            type="button"
            onClick={() => onDelete(assetUuid)}
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

export default DeleteAssetModal;
