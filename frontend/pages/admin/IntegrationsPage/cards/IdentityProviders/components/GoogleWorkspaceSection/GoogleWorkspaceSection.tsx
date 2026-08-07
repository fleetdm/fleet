import React, { useState, useCallback, useEffect } from "react";
import { useQueryClient } from "react-query";

import { IInputFieldParseTarget } from "interfaces/form_field";
import { IConfig } from "interfaces/config";
import configAPI from "services/entities/config";
import { UNCHANGED_PASSWORD_API_RESPONSE } from "utilities/constants";

import { notify } from "components/ToastNotification";
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import Card from "components/Card";
import PageDescription from "components/PageDescription";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import SettingsSection from "pages/admin/components/SettingsSection";

const API_KEY_JSON_PLACEHOLDER = `{
  "type": "service_account",
  "project_id": "fleet-idp-sync",
  "private_key_id": "<private key id>",
  "private_key": "-----BEGIN PRIVATE KEY----\\n<private key>\\n-----END PRIVATE KEY-----\\n",
  "client_email": "fleet-idp-sync@fleet-idp-sync.iam.gserviceaccount.com",
  "client_id": "<client id>",
  "token_uri": "https://oauth2.googleapis.com/token",
  "universe_domain": "googleapis.com"
}`;

const isObfuscatedApiKey = (apiKeyJson: Record<string, string>): boolean => {
  if (!apiKeyJson || Object.keys(apiKeyJson).length === 0) {
    return false;
  }
  return Object.values(apiKeyJson).every(
    (value) => value === UNCHANGED_PASSWORD_API_RESPONSE
  );
};

interface IGoogleWorkspaceFormErrors {
  domain?: string | null;
  impersonatedUserEmail?: string | null;
  apiKeyJson?: string | null;
}

interface IGoogleWorkspaceFormData {
  domain: string;
  impersonatedUserEmail: string;
  apiKeyJson: string;
}

type ErrorWithMessage = {
  message: string;
  [key: string]: unknown;
};

const isErrorWithMessage = (error: unknown): error is ErrorWithMessage => {
  return (error as ErrorWithMessage).message !== undefined;
};

const baseClass = "google-workspace-section";

interface IGoogleWorkspaceSectionProps {
  appConfig: IConfig;
}

const GoogleWorkspaceSection = ({
  appConfig,
}: IGoogleWorkspaceSectionProps): JSX.Element => {
  const queryClient = useQueryClient();

  const [formData, setFormData] = useState<IGoogleWorkspaceFormData>({
    domain: "",
    impersonatedUserEmail: "",
    apiKeyJson: "",
  });
  const [isUpdatingSettings, setIsUpdatingSettings] = useState(false);
  const [formErrors, setFormErrors] = useState<IGoogleWorkspaceFormErrors>({});

  // Sync form state from the config prop.
  useEffect(() => {
    const integrations = appConfig?.integrations.google_workspace;
    if (Array.isArray(integrations) && integrations.length > 0) {
      const { domain, impersonated_user_email, api_key_json } = integrations[0];
      setFormData({
        domain,
        impersonatedUserEmail: impersonated_user_email,
        apiKeyJson: isObfuscatedApiKey(api_key_json)
          ? UNCHANGED_PASSWORD_API_RESPONSE
          : JSON.stringify(api_key_json, null, "\t"),
      });
    }
  }, [appConfig]);

  const gomEnabled = appConfig.gitops.gitops_mode_enabled;
  const { apiKeyJson, domain, impersonatedUserEmail } = formData;

  const validateForm = (curFormData: IGoogleWorkspaceFormData) => {
    const errors: IGoogleWorkspaceFormErrors = {};
    const anyFilled =
      !!curFormData.domain ||
      !!curFormData.impersonatedUserEmail ||
      !!curFormData.apiKeyJson;

    // All-or-nothing: if any field is set, all are required.
    if (anyFilled) {
      if (!curFormData.domain) {
        errors.domain = "Primary domain must be completed";
      }
      if (!curFormData.impersonatedUserEmail) {
        errors.impersonatedUserEmail = "Admin email must be completed";
      }
      if (!curFormData.apiKeyJson) {
        errors.apiKeyJson = "API key JSON must be completed";
      }
    }
    if (
      curFormData.apiKeyJson &&
      curFormData.apiKeyJson !== UNCHANGED_PASSWORD_API_RESPONSE
    ) {
      try {
        JSON.parse(curFormData.apiKeyJson);
      } catch (e: unknown) {
        if (isErrorWithMessage(e)) {
          errors.apiKeyJson = e.message.toString();
        } else {
          throw e;
        }
      }
    }
    return errors;
  };

  const onInputChange = useCallback(
    ({ name, value }: IInputFieldParseTarget) => {
      const newFormData = { ...formData, [name]: value };
      setFormData(newFormData);
      setFormErrors(validateForm(newFormData));
    },
    [formData]
  );

  const onFormSubmit = async (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();

    const errors = validateForm(formData);
    setFormErrors(errors);
    if (Object.keys(errors).length > 0) {
      return;
    }

    setIsUpdatingSettings(true);

    try {
      // Disconnect: all fields cleared -> send empty array.
      const isDisconnect = !domain && !impersonatedUserEmail && !apiKeyJson;

      let googleWorkspace: Record<string, unknown>[] = [];
      if (!isDisconnect) {
        const entry: Record<string, unknown> = {
          domain,
          impersonated_user_email: impersonatedUserEmail,
        };
        // Only send api_key_json when it changed (masked value => preserve existing).
        // JSON.parse is inside the try so a malformed key can't throw uncaught.
        if (apiKeyJson && apiKeyJson !== UNCHANGED_PASSWORD_API_RESPONSE) {
          entry.api_key_json = JSON.parse(apiKeyJson);
        }
        googleWorkspace = [entry];
      }

      await configAPI.update({
        integrations: { google_workspace: googleWorkspace },
      });
      notify.success(
        "Successfully saved Google Workspace integration settings."
      );
      await queryClient.invalidateQueries(["config"]);
      await queryClient.invalidateQueries(["scim_details"]);
    } catch (e) {
      notify.error("Could not save Google Workspace integration settings.", {
        response: e,
      });
    } finally {
      setIsUpdatingSettings(false);
    }
  };

  return (
    <SettingsSection title="Google Workspace" className={baseClass}>
      <PageDescription
        content={
          <>
            Configure these settings to populate IdP host vitals from Google
            Workspace. When Google Workspace is connected, Fleet ignores SCIM
            provisioning from other IdPs (e.g Okta, Entra ID).
          </>
        }
        variant="right-panel"
      />
      <form onSubmit={onFormSubmit} autoComplete="off">
        <Card>
          <InputField
            label="API key JSON"
            onChange={onInputChange}
            name="apiKeyJson"
            value={apiKeyJson}
            parseTarget
            type="textarea"
            placeholder={API_KEY_JSON_PLACEHOLDER}
            inputClassName={`${baseClass}__api-key-json`}
            error={formErrors.apiKeyJson}
            disabled={gomEnabled}
            helpText={
              apiKeyJson === UNCHANGED_PASSWORD_API_RESPONSE
                ? "API key is configured. Replace with a new key to update."
                : "Paste the full contents of the service account JSON key file."
            }
          />
          <InputField
            label="Primary domain"
            onChange={onInputChange}
            name="domain"
            value={domain}
            parseTarget
            placeholder="example.com"
            error={formErrors.domain}
            disabled={gomEnabled}
            helpText="Your Google Workspace primary domain."
          />
          <InputField
            label="Admin email to impersonate"
            onChange={onInputChange}
            name="impersonatedUserEmail"
            value={impersonatedUserEmail}
            parseTarget
            placeholder="admin@example.com"
            error={formErrors.impersonatedUserEmail}
            disabled={gomEnabled}
            helpText="A Google Workspace admin the service account impersonates via domain-wide delegation."
          />
          <div className="button-wrap">
            <GitOpsModeTooltipWrapper
              tipOffset={8}
              renderChildren={(disableChildren) => (
                <Button
                  type="submit"
                  disabled={
                    Object.keys(formErrors).length > 0 || disableChildren
                  }
                  className="save-loading"
                  isLoading={isUpdatingSettings}
                >
                  Save
                </Button>
              )}
            />
          </div>
        </Card>
      </form>
    </SettingsSection>
  );
};

export default GoogleWorkspaceSection;
