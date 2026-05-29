import React, { useContext, useState } from "react";

import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";
import configAPI from "services/entities/config";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "delete-entra-client-id-modal";

interface IDeleteEntraClientIdModalProps {
  clientId: string;
  onExit: () => void;
}

const DeleteEntraClientIdModal = ({
  clientId,
  onExit,
}: IDeleteEntraClientIdModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const { setConfig, config } = useContext(AppContext);

  const [isDeleting, setIsDeleting] = useState(false);

  const onDeleteClientId = async () => {
    setIsDeleting(true);

    try {
      const currentClientIds = config?.mdm.windows_entra_client_ids ?? [];
      const updatedClientIds = currentClientIds.filter((id) => id !== clientId);
      const updateData = await configAPI.update({
        mdm: {
          windows_entra_client_ids: updatedClientIds,
        },
      });
      setConfig(updateData);
      renderFlash("success", "Client ID deleted successfully.");
      onExit();
    } catch (err) {
      renderFlash("error", "Couldn't delete client ID. Please try again.");
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <Modal
      className={baseClass}
      title="Delete client ID"
      onExit={onExit}
      width="medium"
      isContentDisabled={isDeleting}
    >
      <p>
        Windows hosts won&apos;t be able to enroll using the Microsoft Entra
        application with this client ID. Your other tenant IDs and client IDs
        are unaffected.
      </p>
      <div className="modal-cta-wrap">
        <Button
          onClick={onDeleteClientId}
          variant="alert"
          isLoading={isDeleting}
        >
          Delete
        </Button>
        <Button onClick={onExit} variant="inverse">
          Cancel
        </Button>
      </div>
    </Modal>
  );
};

export default DeleteEntraClientIdModal;
