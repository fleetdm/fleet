import React, { useState } from "react";

import { useQueryClient } from "react-query";

import {
  LEARN_MORE_ABOUT_BASE_LINK,
  UNCHANGED_PASSWORD_API_RESPONSE,
} from "utilities/constants";
import configAPI from "services/entities/config";
import { getErrorReason } from "interfaces/errors";
import { notify } from "components/ToastNotification";
import { IAppConfigFormProps } from "pages/admin/OrgSettingsPage/cards/constants";

import SettingsSection from "pages/admin/components/SettingsSection";
import PageDescription from "components/PageDescription";
import CustomLink from "components/CustomLink";
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import validUrl from "components/forms/validators/valid_url";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import useGitOpsMode from "hooks/useGitOpsMode";
import { isPremiumTier } from "utilities/permissions/permissions";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import useFormValidation, { IFormErrors } from "hooks/useFormValidation";

const baseClass = "account-provisioning";

interface IFormData {
  tokenUrl: string;
  clientId: string;
  clientSecret: string;
}

const isEmptyFormData = (data: IFormData) => {
  return !data.tokenUrl && !data.clientId && !data.clientSecret;
};

const validate = (rawFormData: IFormData): IFormErrors => {
  const errors: IFormErrors = {};

  const formData: IFormData = {
    tokenUrl: rawFormData.tokenUrl.trim(),
    clientId: rawFormData.clientId.trim(),
    clientSecret: rawFormData.clientSecret.trim(),
  };

  if (isEmptyFormData(formData)) {
    // Clearing a form is a valid state.
    return errors;
  }

  if (!formData.tokenUrl) {
    errors.tokenUrl = "Token URL is required.";
  } else if (!validUrl({ url: formData.tokenUrl, protocols: ["https"] })) {
    errors.tokenUrl =
      "Must be a valid https URL (e.g. https://yourdomain.okta.com/oauth2/v1/token)";
  }

  if (!formData.clientId) {
    errors.clientId = "Client ID is required.";
  }

  if (!formData.clientSecret) {
    errors.clientSecret = "Client secret is required.";
  }

  return errors;
};

const SERVER_ERROR_NAMES: Record<keyof IFormData, string> = {
  tokenUrl: "mdm.apple_account_provisioning.oauth_idp_token_url",
  clientId: "mdm.apple_account_provisioning.oauth_idp_client_id",
  clientSecret: "mdm.apple_account_provisioning.oauth_idp_client_secret",
};

const getServerFieldErrors = (err: unknown): IFormErrors => {
  const errors: IFormErrors = {};
  (Object.keys(SERVER_ERROR_NAMES) as (keyof IFormData)[]).forEach((field) => {
    const reason = getErrorReason(err, {
      nameEquals: SERVER_ERROR_NAMES[field],
    });
    if (reason) {
      errors[field] = reason;
    }
  });
  return errors;
};

const AccountProvisioning = ({ appConfig }: IAppConfigFormProps) => {
  const { gitOpsModeEnabled } = useGitOpsMode();
  const queryClient = useQueryClient();
  const [isUpdating, setIsUpdating] = useState(false);
  const [serverFormErrors, setServerFormErrors] = useState<IFormErrors>({});

  const {
    formData,
    setField,
    validateField,
    getError,
    setFieldError,
    clearFieldError,
    clearErrors,
    handleSubmit,
    isSubmitting,
  } = useFormValidation<IFormData>({
    validate,
    initialFormData: {
      clientId:
        appConfig.mdm.apple_account_provisioning?.oauth_idp_client_id || "",
      clientSecret:
        appConfig.mdm.apple_account_provisioning?.oauth_idp_client_secret || "",
      tokenUrl:
        appConfig.mdm.apple_account_provisioning?.oauth_idp_token_url || "",
    },
    serverErrors: serverFormErrors,
    isSubmitting: isUpdating,
  });

  const onFieldChange = (name: keyof IFormData, value: string) => {
    setField(name, value);

    // Emptying the last field is how the configuration gets cleared, and an
    // empty form is valid, so every required-field error stops applying.
    if (isEmptyFormData({ ...formData, [name]: value })) {
      clearErrors();
      return;
    }

    if (
      name === "tokenUrl" &&
      formData.clientSecret === UNCHANGED_PASSWORD_API_RESPONSE
    ) {
      // The server rejects a token URL change that reuses the stored secret
      // (the secret would be sent to the new, possibly hostile, URL), so clear
      // the masked secret and have the user re-enter it. Same pattern as
      // editing a certificate authority.
      setField("clientSecret", "");
      setFieldError(
        "clientSecret",
        "Client secret must be re-entered when changing the token URL."
      );
    }
  };

  const onSubmit = async (data: IFormData) => {
    const secretToSubmit =
      data.clientSecret === UNCHANGED_PASSWORD_API_RESPONSE
        ? undefined
        : data.clientSecret;

    setIsUpdating(true);
    try {
      await configAPI.update({
        mdm: {
          apple_account_provisioning: {
            oauth_idp_token_url: data.tokenUrl,
            oauth_idp_client_id: data.clientId,
            ...(secretToSubmit !== undefined && {
              oauth_idp_client_secret: secretToSubmit,
            }),
          },
        },
      });
      await queryClient.invalidateQueries(["config"]);
      notify.success("Successfully updated settings.");
    } catch (err) {
      setServerFormErrors(getServerFieldErrors(err));
      const reason = getErrorReason(err);
      notify.error(
        reason
          ? `Failed to update settings: ${reason}`
          : "Failed to update settings.",
        { response: err }
      );
    } finally {
      setIsUpdating(false);
    }
  };

  const render = () => {
    if (!isPremiumTier(appConfig)) {
      return <PremiumFeatureMessage />;
    }

    return (
      <>
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
        <form onSubmit={handleSubmit(onSubmit)}>
          <div
            className={`form ${
              gitOpsModeEnabled ? "disabled-by-gitops-mode" : ""
            }`}
          >
            <InputField
              label="Token URL"
              name="tokenUrl"
              value={formData.tokenUrl}
              onChange={(val) => onFieldChange("tokenUrl", val)}
              onBlur={() => validateField("tokenUrl")}
              onFocus={() => clearFieldError("tokenUrl")}
              error={getError("tokenUrl")}
              disabled={isSubmitting}
              placeholder="https://yourdomain.okta.com/oauth2/v1/token"
              helpText="Your IdP URL for verifying login credentials. For Okta, this is typically https://yourdomain.okta.com/oauth2/v1/token."
            />
            <InputField
              label="Client ID"
              name="clientId"
              value={formData.clientId}
              onChange={(val) => onFieldChange("clientId", val)}
              onBlur={() => validateField("clientId")}
              onFocus={() => clearFieldError("clientId")}
              error={getError("clientId")}
              helpText="In Okta, this will be in the Client Credentials section."
              disabled={isSubmitting}
            />
            <InputField
              type="password"
              label="Client secret"
              name="clientSecret"
              value={formData.clientSecret}
              onChange={(val) => onFieldChange("clientSecret", val)}
              onBlur={() => validateField("clientSecret")}
              onFocus={() => clearFieldError("clientSecret")}
              error={getError("clientSecret")}
              helpText="In Okta, this will be in the Client Credentials section."
              disabled={isSubmitting}
            />
          </div>
          <GitOpsModeTooltipWrapper
            renderChildren={(disableChildren) => (
              <Button
                type="submit"
                disabled={disableChildren || isSubmitting}
                isLoading={isSubmitting}
              >
                Save
              </Button>
            )}
          />
        </form>
      </>
    );
  };

  return (
    <SettingsSection title="Account provisioning" className={baseClass}>
      {render()}
    </SettingsSection>
  );
};

export default AccountProvisioning;
