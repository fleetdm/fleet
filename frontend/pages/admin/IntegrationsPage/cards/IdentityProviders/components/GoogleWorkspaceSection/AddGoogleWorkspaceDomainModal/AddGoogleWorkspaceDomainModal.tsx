import React from "react";
import { useQueryClient } from "react-query";

import Button from "components/buttons/Button";
import InputField from "components/forms/fields/InputField";
import Modal from "components/Modal";
import { notify } from "components/ToastNotification";
import useFormValidation, { IFormErrors } from "hooks/useFormValidation";
import { IConfig } from "interfaces/config";
import { IGlobalGoogleWorkspaceIntegration } from "interfaces/integration";
import configAPI from "services/entities/config";

import buildAdditionalDomainsUpdate from "../helpers";

const baseClass = "add-google-workspace-domain-modal";

interface IAddDomainFormData {
  domain: string;
}

interface IAddGoogleWorkspaceDomainModalProps {
  integration: IGlobalGoogleWorkspaceIntegration;
  onExit: () => void;
}

const AddGoogleWorkspaceDomainModal = ({
  integration,
  onExit,
}: IAddGoogleWorkspaceDomainModalProps) => {
  const queryClient = useQueryClient();

  const validate = ({ domain }: IAddDomainFormData): IFormErrors => {
    const value = domain.trim();
    if (!value) {
      return { domain: "Domain is required" };
    }
    if (/[\s,]/.test(value)) {
      return { domain: "Enter a single domain without spaces or commas" };
    }
    const existing = [
      integration.domain,
      ...(integration.additional_domains ?? []),
    ].map((d) => d.toLowerCase());
    if (existing.includes(value.toLowerCase())) {
      return { domain: "Domain already added" };
    }
    return {};
  };

  const {
    getTextFieldProps,
    handleSubmit,
    isSubmitting,
  } = useFormValidation<IAddDomainFormData>({
    initialFormData: { domain: "" },
    validate,
  });

  const onValidSubmit = async ({ domain }: IAddDomainFormData) => {
    try {
      const updatedConfig: IConfig = await configAPI.update(
        buildAdditionalDomainsUpdate(integration, [
          ...(integration.additional_domains ?? []),
          domain,
        ])
      );
      queryClient.setQueryData(["config"], updatedConfig);
      notify.success("Successfully added domain.");
      onExit();
    } catch (e) {
      notify.error("Couldn't add domain. Please try again.", { response: e });
    }
  };

  return (
    <Modal
      className={baseClass}
      title="Add domain"
      onExit={onExit}
      isContentDisabled={isSubmitting}
    >
      <form autoComplete="off" onSubmit={handleSubmit(onValidSubmit)}>
        <InputField
          {...getTextFieldProps("domain")}
          // A distinct name keeps this input's id from colliding with the
          // section's "Primary domain" field.
          name="additionalDomain"
          label="Domain"
          placeholder="example.org"
          helpText="A secondary domain in the same Google Workspace account. Users whose primary email is in this domain are synced."
        />
        <div className="modal-cta-wrap">
          <Button
            type="submit"
            isLoading={isSubmitting}
            disabled={isSubmitting}
          >
            Add
          </Button>
          <Button onClick={onExit} variant="secondary">
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default AddGoogleWorkspaceDomainModal;
