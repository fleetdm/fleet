import React, { useContext, useState } from "react";

import { AppContext } from "context/app";
import configAPI from "services/entities/config";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import { notify } from "components/ToastNotification";

const baseClass = "delete-entra-tenant-modal";

interface IDeleteEntraTenantModalProps {
  tenantId: string;
  onExit: () => void;
}

const DeleteEntraTenantModal = ({
  tenantId,
  onExit,
}: IDeleteEntraTenantModalProps) => {
  const { setConfig, config } = useContext(AppContext);

  const [isDeleting, setIsDeleting] = useState(false);

  const onDeleteToken = async () => {
    setIsDeleting(true);

    try {
      const currentTenantIds = config?.mdm.windows_entra_tenant_ids ?? [];
      const updatedTenantIds = currentTenantIds.filter((id) => id !== tenantId);
      const updateData = await configAPI.update({
        mdm: {
          windows_entra_tenant_ids: updatedTenantIds,
        },
      });
      setConfig(updateData);
      notify.success("Tenant deleted successfully.");
      onExit();
    } catch (err) {
      notify.error("Couldn't delete tenant. Please try again.", {
        response: err,
      });
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <Modal
      className={baseClass}
      title="Delete tenant"
      onExit={onExit}
      width="medium"
      isContentDisabled={isDeleting}
    >
      <p>
        This will stop both automatic (Autopilot) and manual enrollment by end
        users (<b>Settings &gt; Accounts &gt; Access work or school</b> on
        Windows) from this tenant.
      </p>
      <div className="modal-cta-wrap">
        <Button onClick={onDeleteToken} variant="alert" isLoading={isDeleting}>
          Delete
        </Button>
        <Button onClick={onExit} variant="inverse">
          Cancel
        </Button>
      </div>
    </Modal>
  );
};

export default DeleteEntraTenantModal;
