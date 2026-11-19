import React, { useState } from "react";
import { useQueryClient } from "react-query";

import Button from "components/buttons/Button";
import InputField from "components/forms/fields/InputField";
import Modal from "components/Modal";
import { notify } from "components/ToastNotification";
import { IGlobalGoogleWorkspaceIntegration } from "interfaces/integration";
import configAPI from "services/entities/config";

import buildAdditionalDomainsUpdate from "../helpers";

const baseClass = "add-google-workspace-domain-modal";

interface IAddGoogleWorkspaceDomainModalProps {
  integration: IGlobalGoogleWorkspaceIntegration;
  onExit: () => void;
}

const validateDomain = (
  domain: string,
  integration: IGlobalGoogleWorkspaceIntegration
): string | null => {
  if (!domain) {
    return "Domain is required";
  }
  if (/[\s,]/.test(domain)) {
    return "Enter a single domain without spaces or commas";
  }
  const existing = [
    integration.domain,
    ...(integration.additional_domains ?? []),
  ].map((d) => d.toLowerCase());
  if (existing.includes(domain.toLowerCase())) {
    return "Domain already added";
  }
  return null;
};

const AddGoogleWorkspaceDomainModal = ({
  integration,
  onExit,
}: IAddGoogleWorkspaceDomainModalProps) => {
  const queryClient = useQueryClient();
  const [domain, setDomain] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isAdding, setIsAdding] = useState(false);

  const onChange = (value: string) => {
    setDomain(value);
    setError(value ? validateDomain(value.trim(), integration) : null);
  };

  const onAdd = async () => {
    const trimmed = domain.trim();
    const validationError = validateDomain(trimmed, integration);
    if (validationError) {
      setError(validationError);
      return;
    }

    setIsAdding(true);
    try {
      await configAPI.update(
        buildAdditionalDomainsUpdate(integration, [
          ...(integration.additional_domains ?? []),
          trimmed,
        ])
      );
      await queryClient.invalidateQueries(["config"]);
      notify.success("Successfully added domain.");
      onExit();
    } catch (e) {
      notify.error("Couldn't add domain. Please try again.", { response: e });
    } finally {
      setIsAdding(false);
    }
  };

  return (
    <Modal
      className={baseClass}
      title="Add domain"
      onExit={onExit}
      isContentDisabled={isAdding}
    >
      <InputField
        label="Domain"
        name="additionalDomain"
        placeholder="example.org"
        value={domain}
        onChange={onChange}
        error={error}
        helpText="A secondary domain in the same Google Workspace account. Users whose primary email is in this domain are synced."
      />
      <div className="modal-cta-wrap">
        <Button
          onClick={onAdd}
          disabled={!domain || !!error}
          isLoading={isAdding}
        >
          Add
        </Button>
        <Button onClick={onExit} variant="secondary">
          Cancel
        </Button>
      </div>
    </Modal>
  );
};

export default AddGoogleWorkspaceDomainModal;
