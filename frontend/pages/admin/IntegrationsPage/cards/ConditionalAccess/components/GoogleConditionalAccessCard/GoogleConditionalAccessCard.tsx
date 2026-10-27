import React, { useContext, useState } from "react";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import InputField from "components/forms/fields/InputField";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import { notify } from "components/ToastNotification";
import { AppContext } from "context/app";
import useFormValidation, { IFormErrors } from "hooks/useFormValidation";
import {
  IConfig,
  isGoogleConditionalAccessConfigured,
} from "interfaces/config";
import { IApiError } from "interfaces/errors";
import configAPI from "services/entities/config";
import {
  LEARN_MORE_ABOUT_BASE_LINK,
  UNCHANGED_PASSWORD_API_RESPONSE,
} from "utilities/constants";

import SectionCard from "../../../MdmSettings/components/SectionCard";

const baseClass = "google-conditional-access-card";

const INTEGRATION_ERROR_NAME = "integrations.google_conditional_access";

// google_conditional_access_integrations.customer_id is VARCHAR(64).
const CUSTOMER_ID_MAX_LENGTH = 64;

const API_KEY_JSON_PLACEHOLDER = `{
  "type": "service_account",
  "project_id": "<project id>",
  "private_key_id": "<private key id>",
  "private_key": "-----BEGIN PRIVATE KEY----\\n<private key>\\n-----END PRIVATE KEY-----\\n",
  "client_email": "<name>@<project id>.iam.gserviceaccount.com",
  "client_id": "<client id>",
  "token_uri": "https://oauth2.googleapis.com/token"
}`;

interface IFormData {
  customer_id: string;
  admin_email: string;
  api_key_json: string;
}

const parseApiKeyJson = (value: string): Record<string, string> | null => {
  try {
    const parsed = JSON.parse(value);
    return parsed && typeof parsed === "object" && !Array.isArray(parsed)
      ? parsed
      : null;
  } catch {
    return null;
  }
};

const validate = (data: IFormData): IFormErrors => {
  const errors: IFormErrors = {};
  if (!data.customer_id.trim()) {
    errors.customer_id = "Enter your Google customer ID";
  }
  if (!data.admin_email.trim()) {
    errors.admin_email = "Enter an admin email";
  }
  if (
    data.api_key_json !== UNCHANGED_PASSWORD_API_RESPONSE &&
    !parseApiKeyJson(data.api_key_json)
  ) {
    errors.api_key_json = "Paste the service account key JSON";
  }
  return errors;
};

const FIELD_ERRORS: Record<keyof IFormData, string> = {
  customer_id: "Enter a valid Google customer ID",
  admin_email: "Enter a valid admin email",
  api_key_json: "Paste a valid service account key JSON",
};

const getApiErrors = (e: unknown) => {
  const apiErrors = (e as { data?: IApiError } | undefined)?.data?.errors;
  return Array.isArray(apiErrors) ? apiErrors : [];
};

// Field errors are named e.g. "integrations.google_conditional_access.api_key_json.private_key".
const parseFieldErrors = (e: unknown): IFormErrors | null => {
  const errors: IFormErrors = {};
  getApiErrors(e).forEach(({ name }) => {
    const field = (Object.keys(FIELD_ERRORS) as (keyof IFormData)[]).find(
      (f) =>
        name === `${INTEGRATION_ERROR_NAME}.${f}` ||
        name?.startsWith(`${INTEGRATION_ERROR_NAME}.${f}.`)
    );
    if (field) {
      errors[field] = FIELD_ERRORS[field];
    }
  });
  return Object.keys(errors).length ? errors : null;
};

interface IGoogleConditionalAccessFormProps {
  config: IConfig | null;
  onConnected: (config: IConfig) => void;
}

const GoogleConditionalAccessForm = ({
  config,
  onConnected,
}: IGoogleConditionalAccessFormProps) => {
  const [serverErrors, setServerErrors] = useState<IFormErrors | null>(null);

  // An entry can exist without a completed connection; prefill it and keep the
  // stored key unless the admin pastes a new one.
  const existing = config?.integrations.google_conditional_access?.[0];

  const {
    formData,
    setField,
    getError,
    clearFieldError,
    validateField,
    handleSubmit,
    isSubmitting,
  } = useFormValidation<IFormData>({
    initialFormData: {
      customer_id: existing?.customer_id ?? "",
      admin_email: existing?.admin_email ?? "",
      api_key_json: existing ? UNCHANGED_PASSWORD_API_RESPONSE : "",
    },
    validate,
    serverErrors,
  });

  const onValidSubmit = async (data: IFormData) => {
    setServerErrors(null);
    const entry: Record<string, unknown> = {
      customer_id: data.customer_id,
      admin_email: data.admin_email,
    };
    if (data.api_key_json !== UNCHANGED_PASSWORD_API_RESPONSE) {
      entry.api_key_json = parseApiKeyJson(data.api_key_json);
    }
    try {
      const updatedConfig = await configAPI.update({
        integrations: { google_conditional_access: [entry] },
      });
      notify.success("Successfully connected to Google.");
      onConnected(updatedConfig);
    } catch (e) {
      const fieldErrors = parseFieldErrors(e);
      if (fieldErrors) {
        setServerErrors(fieldErrors);
        return;
      }
      const integrationError = getApiErrors(e).find(
        (err) => err?.name === INTEGRATION_ERROR_NAME
      );
      if (integrationError?.reason) {
        notify.error(integrationError.reason);
      } else {
        notify.error("Couldn't connect to Google. Please try again.", {
          response: e,
        });
      }
    }
  };

  const renderField = (
    name: keyof IFormData,
    props: Partial<React.ComponentProps<typeof InputField>>
  ) => (
    <GitOpsModeTooltipWrapper
      isInputField
      renderChildren={(disableChildren) => (
        <InputField
          {...props}
          name={name}
          value={formData[name]}
          onChange={(value: string) => setField(name, value)}
          onFocus={() => clearFieldError(name)}
          onBlur={() => validateField(name)}
          error={getError(name)}
          disabled={isSubmitting || disableChildren}
        />
      )}
    />
  );

  return (
    <form
      className={`${baseClass}__form`}
      onSubmit={handleSubmit(onValidSubmit)}
      autoComplete="off"
    >
      {renderField("customer_id", {
        label: "Google customer ID",
        placeholder: "C01234abc",
        helpText:
          "Find it in the Google Admin console under Account > Account settings.",
        inputOptions: { maxLength: CUSTOMER_ID_MAX_LENGTH },
      })}
      {renderField("admin_email", {
        label: "Admin email",
        placeholder: "fleet-admin@example.com",
        helpText:
          "Fleet marks devices as managed as this Google Workspace admin.",
      })}
      {renderField("api_key_json", {
        label: "Service account key",
        type: "textarea",
        placeholder: API_KEY_JSON_PLACEHOLDER,
        inputClassName: `${baseClass}__api-key-json`,
        helpText:
          formData.api_key_json === UNCHANGED_PASSWORD_API_RESPONSE
            ? "Service account key is configured. Replace with a new key to update."
            : "Paste the full contents of the service account JSON key file.",
      })}
      <div className={`${baseClass}__button-wrap`}>
        <GitOpsModeTooltipWrapper
          renderChildren={(disableChildren) => (
            <Button
              type="submit"
              isLoading={isSubmitting}
              disabled={isSubmitting || disableChildren}
            >
              Save
            </Button>
          )}
        />
      </div>
    </form>
  );
};

export interface IGoogleConditionalAccessCardProps {
  onDelete: () => void;
}

const GoogleConditionalAccessCard = ({
  onDelete,
}: IGoogleConditionalAccessCardProps) => {
  const { config, setConfig } = useContext(AppContext);

  if (isGoogleConditionalAccessConfigured(config)) {
    const integration = config?.integrations.google_conditional_access?.[0];
    const customerId =
      integration?.customer_id ||
      config?.conditional_access?.google_customer_id;
    return (
      <SectionCard
        className={baseClass}
        iconName="success"
        cta={
          <GitOpsModeTooltipWrapper
            renderChildren={(disableChildren) => (
              <Button
                variant="subdued"
                onClick={onDelete}
                icon="trash"
                iconPosition="right"
                disabled={disableChildren}
              >
                Delete
              </Button>
            )}
          />
        }
      >
        <span>Google customer ID: {customerId}</span>
        <span>Admin email: {integration?.admin_email}</span>
      </SectionCard>
    );
  }

  return (
    <SectionCard
      className={`${baseClass} ${baseClass}--not-connected`}
      header="Google"
    >
      <p className={`${baseClass}__description`}>
        Mark company-owned iPhones, iPads, and Android hosts as managed in
        Google, so Google Context-Aware Access can block sign-in from hosts that
        aren&apos;t managed by Fleet.{" "}
        <CustomLink
          url={`${LEARN_MORE_ABOUT_BASE_LINK}/google-conditional-access`}
          text="Learn more"
          newTab
        />
      </p>
      <GoogleConditionalAccessForm
        config={config}
        onConnected={(updatedConfig) => setConfig(updatedConfig)}
      />
    </SectionCard>
  );
};

export default GoogleConditionalAccessCard;
