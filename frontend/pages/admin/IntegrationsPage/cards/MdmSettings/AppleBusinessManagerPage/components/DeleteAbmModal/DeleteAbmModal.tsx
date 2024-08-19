import React, { useCallback, useContext, useState } from "react";

import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { NotificationContext } from "context/notification";
import { IMdmAbmToken } from "interfaces/mdm";

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

  const onClickConfirm = useCallback(() => {
    setIsDeleting(true);
  }, []);

  return (
    <Modal title="Delete ABM" className={baseClass} onExit={onCancel}>
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
            onClick={onClickConfirm}
            disabled={isDeleting}
            isLoading={isDeleting}
          >
            Disable
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
