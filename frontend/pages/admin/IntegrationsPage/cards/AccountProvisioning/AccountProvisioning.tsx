import React, { useEffect, useState } from "react";

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
import { IInputFieldParseTarget } from "interfaces/form_field";
import Button from "components/buttons/Button";
import validUrl from "components/forms/validators/valid_url";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import useGitOpsMode from "hooks/useGitOpsMode";
import { isPremiumTier } from "utilities/permissions/permissions";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";

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
  const [formData, setFormData] = useState<IFormData>({
    tokenUrl: "",
    clientId: "",
    clientSecret: "",
  });
  const [formErrors, setFormErrors] = useState<IFormErrors>({});

  useEffect(() => {
    const provisioning = appConfig.mdm.apple_account_provisioning;
    if (provisioning) {
      setFormData({
        tokenUrl: provisioning.oauth_idp_token_url,
        clientId: provisioning.oauth_idp_client_id,
        clientSecret: provisioning.oauth_idp_client_secret,
      });
    }
  }, [appConfig]);

  const onInputChange = ({ name, value }: IInputFieldParseTarget) => {
    const newFormData = { ...formData, [name]: value };

    // The server rejects a token URL change that reuses the stored secret
    // (the secret would be sent to the new, possibly hostile, URL), so clear
    // the masked secret and have the user re-enter it. Same pattern as
    // editing a certificate authority.
    const secretCleared =
      name === "tokenUrl" &&
      formData.clientSecret === UNCHANGED_PASSWORD_API_RESPONSE;
    if (secretCleared) {
      newFormData.clientSecret = "";
    }

    setFormData(newFormData);
    setFormErrors((prev) => {
      const next = { ...prev };
      if (secretCleared) {
        next.clientSecret =
          "Client secret must be re-entered when changing the token URL.";
      }
      // only update errors for fields that already have an error
      if (prev[name as keyof IFormErrors]) {
        next[name as keyof IFormErrors] = validate(newFormData)[
          name as keyof IFormErrors
        ];
      }
      return next;
    });
  };

  const onInputBlur = (field: keyof IFormData) => () => {
    const newErrors = validate(formData);
    setFormErrors((prev) => ({ ...prev, [field]: newErrors[field] }));
  };

  const onSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const errors = validate(formData);
    if (Object.keys(errors).length > 0) {
      setFormErrors(errors);
      return;
    }

    const secretToSubmit =
      formData.clientSecret === UNCHANGED_PASSWORD_API_RESPONSE
        ? undefined
        : formData.clientSecret;

    setIsUpdating(true);
    try {
      await configAPI.update({
        mdm: {
          apple_account_provisioning: {
            oauth_idp_token_url: formData.tokenUrl,
            oauth_idp_client_id: formData.clientId,
            ...(secretToSubmit !== undefined && {
              oauth_idp_client_secret: secretToSubmit,
            }),
          },
        },
      });
      await queryClient.invalidateQueries(["config"]);
      notify.success("Successfully updated settings.");
    } catch (err) {
      setFormErrors((prev) => ({ ...prev, ...getServerFieldErrors(err) }));
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
        <form onSubmit={onSubmit}>
          <div
            className={`form ${
              gitOpsModeEnabled ? "disabled-by-gitops-mode" : ""
            }`}
          >
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
              type="password"
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
              <Button
                type="submit"
                disabled={disableChildren}
                isLoading={isUpdating}
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
