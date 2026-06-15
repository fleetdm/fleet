import React, { useState } from "react";

import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import SettingsSection from "pages/admin/components/SettingsSection";
import PageDescription from "components/PageDescription";
import CustomLink from "components/CustomLink";
import InputField from "components/forms/fields/InputField";
import { IInputFieldParseTarget } from "interfaces/form_field";
import Button from "components/buttons/Button";
import validUrl from "components/forms/validators/valid_url";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import useGitOpsMode from "hooks/useGitOpsMode";

const baseClass = "account-provisioning";

interface IFormData {
  tokenUrl: string;
  clientId: string;
  clientSecret: string;
}

interface IFormErrors {
  tokenUrl?: string | null;
  clientId?: string | null;
  clientSecret?: string | null;
}

const validate = (formData: IFormData): IFormErrors => {
  const errors: IFormErrors = {};

  if (!formData.tokenUrl) {
    errors.tokenUrl = "Token URL is required.";
  } else if (
    !validUrl({ url: formData.tokenUrl, protocols: ["http", "https"] })
  ) {
    errors.tokenUrl =
      "Must be a valid URL (e.g. https://yourdomain.okta.com/oauth2/v1/token)";
  }

  if (!formData.clientId) {
    errors.clientId = "Client ID is required.";
  }

  if (!formData.clientSecret) {
    errors.clientSecret = "Client secret is required.";
  }

  return errors;
};

const AccountProvisioning = () => {
  const { gitOpsModeEnabled } = useGitOpsMode();
  const [formData, setFormData] = useState<IFormData>({
    tokenUrl: "",
    clientId: "",
    clientSecret: "",
  });
  const [formErrors, setFormErrors] = useState<IFormErrors>({});

  const onInputChange = ({ name, value }: IInputFieldParseTarget) => {
    const newFormData = { ...formData, [name]: value };
    setFormData(newFormData);
    // only update errors for fields that already have an error
    if (formErrors[name as keyof IFormErrors]) {
      const newErrors = validate(newFormData);
      setFormErrors((prev) => ({
        ...prev,
        [name]: newErrors[name as keyof IFormErrors],
      }));
    }
  };

  const onInputBlur = (field: keyof IFormData) => () => {
    const newErrors = validate(formData);
    setFormErrors((prev) => ({ ...prev, [field]: newErrors[field] }));
  };

  const onSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const errors = validate(formData);
    if (Object.keys(errors).length > 0) {
      setFormErrors(errors);
    }
    // TODO: submit to API
  };

  return (
    <SettingsSection title="Account provisioning" className={baseClass}>
      <PageDescription
        variant="right-panel"
        content={
          <>
            Create and sync macOS accounts using IdP credentials with any IdP
            that supports OAuth ROPG (Okta){" "}
            <CustomLink
              newTab
              url={`${LEARN_MORE_ABOUT_BASE_LINK}/idp-account-sync`}
              text="Learn more"
            />
          </>
        }
      />
      <form onSubmit={onSubmit}>
        <div className={`form ${gitOpsModeEnabled ? "disabled-by-gitops-mode" : ""}`}>
          <InputField
            label="Token URL"
            name="tokenUrl"
            value={formData.tokenUrl}
            onChange={onInputChange}
            onBlur={onInputBlur("tokenUrl")}
            parseTarget
            placeholder="https://yourdomain.okta.com/oauth2/v1/token"
            error={formErrors.tokenUrl}
            helpText="Your IdP URL for verifying login credentials. For Okta, this is typically https://yourdomain.okta.com/oauth2/v1/token."
          />
          <InputField
            label="Client ID"
            name="clientId"
            value={formData.clientId}
            onChange={onInputChange}
            onBlur={onInputBlur("clientId")}
            parseTarget
            error={formErrors.clientId}
            helpText="In Okta, this will be in the Client Credentials section."
          />
          <InputField
            label="Client secret"
            name="clientSecret"
            value={formData.clientSecret}
            onChange={onInputChange}
            onBlur={onInputBlur("clientSecret")}
            parseTarget
            error={formErrors.clientSecret}
            helpText="In Okta, this will be in the Client Credentials section."
          />
        </div>
        <GitOpsModeTooltipWrapper
          renderChildren={(disableChildren) => (
            <Button type="submit" disabled={disableChildren}>
              Save
            </Button>
          )}
        />
      </form>
    </SettingsSection>
  );
};

export default AccountProvisioning;
