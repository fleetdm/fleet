import React, { useContext, useEffect, useRef, useState } from "react";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import InputField from "components/forms/fields/InputField";
import { isValidEmail } from "components/forms/validators";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Modal from "components/Modal";
import { notify } from "components/ToastNotification";
import { AppContext } from "context/app";
import useFormValidation, { IFormErrors } from "hooks/useFormValidation";
import useGitOpsMode from "hooks/useGitOpsMode";
import { IConfig } from "interfaces/config";
import { splitServerErrors } from "interfaces/errors";
import conditionalAccessAPI, {
  IGoogleConditionalAccessFormData,
} from "services/entities/conditional_access";
import configAPI from "services/entities/config";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

const baseClass = "google-conditional-access-modal";

const CUSTOMER_ID = "customer_id";
const ADMIN_EMAIL = "admin_email";
const SERVICE_ACCOUNT_KEY = "service_account_key";

// Matches the server's check in server/fleet/conditional_access_google.go.
const GOOGLE_CUSTOMER_ID_REGEX = /^C[0-9A-Za-z]{6,32}$/;

interface IFormData {
  [CUSTOMER_ID]: string;
  [ADMIN_EMAIL]: string;
  [SERVICE_ACCOUNT_KEY]: string;
}

const SERVICE_ACCOUNT_KEY_PLACEHOLDER = `{
  "type": "service_account",
  "project_id": "fleet-conditional-access",
  "private_key_id": "<private key id>",
  "private_key": "-----BEGIN PRIVATE KEY-----\\n<private key>\\n-----END PRIVATE KEY-----\\n",
  "client_email": "fleet@fleet-conditional-access.iam.gserviceaccount.com",
  "client_id": "<client id>",
  "token_uri": "https://oauth2.googleapis.com/token",
  "universe_domain": "googleapis.com"
}`;

const SERVER_ERROR_NAMES: Record<keyof IFormData, string> = {
  [CUSTOMER_ID]: CUSTOMER_ID,
  [ADMIN_EMAIL]: ADMIN_EMAIL,
  [SERVICE_ACCOUNT_KEY]: SERVICE_ACCOUNT_KEY,
};

const parseServiceAccountKey = (
  value: string
): Record<string, unknown> | null => {
  try {
    const parsed: unknown = JSON.parse(value);
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>;
    }
  } catch {
    // Reported as a validation error below.
  }
  return null;
};

const validate = (data: IFormData, keyRequired: boolean): IFormErrors => {
  const errors: IFormErrors = {};

  if (!data[CUSTOMER_ID].trim()) {
    errors[CUSTOMER_ID] = "Enter Google customer ID";
  } else if (!GOOGLE_CUSTOMER_ID_REGEX.test(data[CUSTOMER_ID].trim())) {
    errors[CUSTOMER_ID] = "Enter a valid Google customer ID";
  }

  if (!data[ADMIN_EMAIL].trim()) {
    errors[ADMIN_EMAIL] = "Enter admin email";
  } else if (!isValidEmail(data[ADMIN_EMAIL].trim())) {
    errors[ADMIN_EMAIL] = "Enter a valid email address";
  }

  if (!data[SERVICE_ACCOUNT_KEY].trim()) {
    if (keyRequired) {
      errors[SERVICE_ACCOUNT_KEY] = "Paste the service account JSON";
    }
  } else if (!parseServiceAccountKey(data[SERVICE_ACCOUNT_KEY])) {
    errors[SERVICE_ACCOUNT_KEY] = "Paste valid service account JSON";
  }

  return errors;
};

export interface IGoogleConditionalAccessModalProps {
  onCancel: () => void;
  onSuccess: (updatedConfig: IConfig) => void;
}

const GoogleConditionalAccessModal = ({
  onCancel,
  onSuccess,
}: IGoogleConditionalAccessModalProps) => {
  const { config, setConfig } = useContext(AppContext);
  const { gitOpsModeEnabled } = useGitOpsMode();
  const [serverErrors, setServerErrors] = useState<IFormErrors | null>(null);

  const savedSettings = config?.integrations?.google_conditional_access;
  const isEditing = !!savedSettings?.customer_id;
  // In GitOps mode the customer ID and admin email come from the YAML, but the
  // key can still be uploaded here: a YAML without `api_key_json` keeps it.
  const canSubmitInGitOpsMode = isEditing;
  const keyConfigured = !!config?.conditional_access?.google
    ?.service_account_key_configured;

  // Cancel doesn't abort the request, so a late result must not toast.
  const isMountedRef = useRef(true);
  useEffect(() => {
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  const {
    getTextFieldProps,
    handleSubmit,
    isSubmitting,
  } = useFormValidation<IFormData>({
    initialFormData: {
      [CUSTOMER_ID]: savedSettings?.customer_id ?? "",
      [ADMIN_EMAIL]: savedSettings?.admin_email ?? "",
      [SERVICE_ACCOUNT_KEY]: "",
    },
    // An edit keeps the stored key when none is pasted.
    validate: (data) => validate(data, !(isEditing && keyConfigured)),
    serverErrors,
  });

  const saveSettings = (
    data: IFormData,
    key: Record<string, unknown> | null
  ) => {
    if (!isEditing && key) {
      return conditionalAccessAPI.connectGoogleConditionalAccess({
        customer_id: data[CUSTOMER_ID],
        admin_email: data[ADMIN_EMAIL],
        service_account_key: key,
      });
    }
    // Only what changed, so the server keeps the rest, including the stored
    // key. An empty PATCH retries a connection that didn't complete.
    const changes: Partial<IGoogleConditionalAccessFormData> = {};
    if (data[CUSTOMER_ID] !== savedSettings?.customer_id) {
      changes.customer_id = data[CUSTOMER_ID];
    }
    if (data[ADMIN_EMAIL] !== savedSettings?.admin_email) {
      changes.admin_email = data[ADMIN_EMAIL];
    }
    if (key) {
      changes.service_account_key = key;
    }
    return conditionalAccessAPI.updateGoogleConditionalAccess(changes);
  };

  const onValidSubmit = async (data: IFormData) => {
    const key = data[SERVICE_ACCOUNT_KEY]
      ? parseServiceAccountKey(data[SERVICE_ACCOUNT_KEY])
      : null;
    setServerErrors(null);
    try {
      await saveSettings(data, key);
      const updatedConfig = await configAPI.loadAll();
      if (!isMountedRef.current) {
        setConfig(updatedConfig);
        return;
      }
      notify.success("Successfully connected to Google.");
      onSuccess(updatedConfig);
    } catch (e) {
      const { fieldErrors, otherReasons } = splitServerErrors(
        e,
        SERVER_ERROR_NAMES
      );
      const failedValidation = Object.keys(fieldErrors).length > 0;
      // useFormValidation shows field errors inline and toasts them, so only
      // toast here what has no field to show on.
      let toast: string | undefined;
      const keyError = fieldErrors[SERVICE_ACCOUNT_KEY];
      if (keyError && !key) {
        // The server reports a failed delegation check on the key whatever
        // caused it, which would land on a field the admin left empty.
        delete fieldErrors[SERVICE_ACCOUNT_KEY];
        toast = keyError;
      } else if (otherReasons.length > 0) {
        toast = `Couldn't connect to Google: ${otherReasons.join(" ")}`;
      } else if (!failedValidation) {
        toast = "Couldn't connect to Google. Please try again.";
      }
      if (isMountedRef.current) {
        setServerErrors(fieldErrors);
      }
      if (!failedValidation) {
        // The server saves the settings before registering, so a failure past
        // validation can leave a failed integration for the card to show.
        try {
          setConfig(await configAPI.loadAll());
        } catch {
          // The original failure is reported below.
        }
      }
      if (toast && isMountedRef.current) {
        notify.error(toast, { response: e });
      }
    }
  };

  const renderSaveButton = (disabledByGitOps: boolean) => (
    <Button
      type="submit"
      disabled={isSubmitting || disabledByGitOps}
      isLoading={isSubmitting}
    >
      Save
    </Button>
  );

  return (
    <Modal
      title="Google conditional access"
      onExit={onCancel}
      className={baseClass}
      width="large"
    >
      <form onSubmit={handleSubmit(onValidSubmit)} autoComplete="off">
        <p>
          Mark company-owned iPhones, iPads, and Android hosts as managed in
          Google, so Google Context-Aware Access can block sign-in from hosts
          that aren&apos;t managed by Fleet.{" "}
          <CustomLink
            url={`${LEARN_MORE_ABOUT_BASE_LINK}/google-conditional-access`}
            text="Learn more"
            newTab
          />
        </p>
        <GitOpsModeTooltipWrapper
          isInputField
          renderChildren={(disableChildren) => (
            <InputField
              label="Google customer ID"
              helpText={
                <>
                  You can find this in the Google Admin console under{" "}
                  <b>Account</b> &gt; <b>Account settings</b>.
                </>
              }
              {...getTextFieldProps(CUSTOMER_ID)}
              disabled={isSubmitting || disableChildren}
            />
          )}
        />
        <GitOpsModeTooltipWrapper
          isInputField
          renderChildren={(disableChildren) => (
            <InputField
              label="Admin email"
              helpText="Fleet marks devices as managed using this Google Workspace admin account."
              {...getTextFieldProps(ADMIN_EMAIL)}
              disabled={isSubmitting || disableChildren}
            />
          )}
        />
        <InputField
          type="textarea"
          label="Service account JSON"
          helpText={
            keyConfigured
              ? "Service account key is configured. Paste a new key to replace it."
              : "Paste the full contents of the JSON file downloaded when creating your service account API key."
          }
          placeholder={SERVICE_ACCOUNT_KEY_PLACEHOLDER}
          inputClassName={`${baseClass}__service-account-key`}
          {...getTextFieldProps(SERVICE_ACCOUNT_KEY)}
          disabled={
            isSubmitting || (gitOpsModeEnabled && !canSubmitInGitOpsMode)
          }
        />
        <div className="modal-cta-wrap">
          {canSubmitInGitOpsMode ? (
            renderSaveButton(false)
          ) : (
            <GitOpsModeTooltipWrapper
              renderChildren={(disableChildren) =>
                renderSaveButton(!!disableChildren)
              }
            />
          )}
          <Button onClick={onCancel} variant="secondary">
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default GoogleConditionalAccessModal;
