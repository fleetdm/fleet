import React, { useState } from "react";
import { useQueryClient } from "react-query";

import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { notify } from "components/ToastNotification";
import { IConfig } from "interfaces/config";
import { IGlobalGoogleWorkspaceIntegration } from "interfaces/integration";
import configAPI from "services/entities/config";

import {
  buildAdditionalDomainsUpdate,
  updateCachedIntegrations,
} from "../helpers";

const baseClass = "delete-google-workspace-domain-modal";

interface IDeleteGoogleWorkspaceDomainModalProps {
  integration: IGlobalGoogleWorkspaceIntegration;
  domain: string;
  onExit: () => void;
}

const DeleteGoogleWorkspaceDomainModal = ({
  integration,
  domain,
  onExit,
}: IDeleteGoogleWorkspaceDomainModalProps) => {
  const queryClient = useQueryClient();
  const [isDeleting, setIsDeleting] = useState(false);

  const onDelete = async () => {
    setIsDeleting(true);
    try {
      const updatedConfig: IConfig = await configAPI.update(
        buildAdditionalDomainsUpdate(
          integration,
          (integration.additional_domains ?? []).filter((d) => d !== domain)
        )
      );
      updateCachedIntegrations(queryClient, updatedConfig);
      notify.success("Domain deleted successfully.");
      onExit();
    } catch (e) {
      notify.error("Couldn't delete domain. Please try again.", {
        response: e,
      });
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <Modal
      className={baseClass}
      title="Delete domain"
      onExit={onExit}
      width="medium"
      isContentDisabled={isDeleting}
    >
      <p>
        Fleet will stop syncing users and groups from <b>{domain}</b>. Their IdP
        host vitals are removed on the next sync.
      </p>
      <div className="modal-cta-wrap">
        <Button onClick={onDelete} variant="alert" isLoading={isDeleting}>
          Delete
        </Button>
        <Button onClick={onExit} variant="secondary">
          Cancel
        </Button>
      </div>
    </Modal>
  );
};

export default DeleteGoogleWorkspaceDomainModal;
