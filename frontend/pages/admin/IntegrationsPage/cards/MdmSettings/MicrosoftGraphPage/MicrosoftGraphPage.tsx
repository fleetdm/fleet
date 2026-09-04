import React, { useContext, useEffect, useState } from "react";
import { useQuery } from "react-query";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { UNCHANGED_PASSWORD_API_RESPONSE } from "utilities/constants";
import { IInputFieldParseTarget } from "interfaces/form_field";
import { equalsIgnoreCase } from "utilities/strings/stringUtils";
import {
  IMicrosoftGraphCredential,
  IMicrosoftGraphCredentialFormData,
} from "interfaces/microsoft_graph_credential";
import microsoftGraphCredentialsAPI, {
  IGetMicrosoftGraphCredentialsResponse,
} from "services/entities/microsoft_graph_credentials";
import { getErrorReason } from "interfaces/errors";

import BackButton from "components/BackButton";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import DataError from "components/DataError";
import DataSet from "components/DataSet";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Icon from "components/Icon";
import { HumanTimeDiffWithDateTip } from "components/HumanTimeDiffWithDateTip";
import InputField from "components/forms/fields/InputField";
import isUUID from "components/forms/validators/valid_uuid";
import MainContent from "components/MainContent";
import PageDescription from "components/PageDescription";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import Spinner from "components/Spinner";
import TooltipWrapper from "components/TooltipWrapper";
import { notify } from "components/ToastNotification";

import DeleteMicrosoftGraphCredentialModal from "./DeleteMicrosoftGraphCredentialModal";

const baseClass = "microsoft-graph-page";

// The two IDs are Entra GUIDs held in varchar(255) columns. The secret bound below is only a sanity limit, far above any secret Entra issues.
const ID_MAX_LENGTH = 255;
const SECRET_MAX_LENGTH = 1024;

// The API never returns a stored secret, so the field shows this placeholder to signal one exists. It doubles as the
// "unchanged" sentinel: the secret is only sent when the value differs from it. Aliased to the shared constant so the
// two can't drift, since the API would accept the mask on a write.
const STORED_SECRET_PLACEHOLDER = UNCHANGED_PASSWORD_API_RESPONSE;

type ICredentialField = "tenantId" | "clientId" | "clientSecret";

type IFormData = Record<ICredentialField, string>;

type IFormErrors = Partial<Record<ICredentialField, string>>;

const SECRET_REQUIRED_ERROR = "Enter a client secret";

const CREDENTIAL_FIELDS: ICredentialField[] = [
  "tenantId",
  "clientId",
  "clientSecret",
];

// The API lower-cases both IDs before comparing, so the UI must too.
const identityMatchesStored = (
  ids: Pick<IFormData, "tenantId" | "clientId">,
  stored: IMicrosoftGraphCredential
) =>
  equalsIgnoreCase(ids.tenantId.trim(), stored.tenant_id) &&
  equalsIgnoreCase(ids.clientId.trim(), stored.client_id);

// The API reports per-field problems under these names. Anything it reports under the bare `microsoft_graph_credentials`
// key is a whole-credential failure (verification, licensing, missing private key) with no field to attach it to.
const SERVER_ERROR_NAMES: Record<ICredentialField, string> = {
  tenantId: "microsoft_graph_credentials.tenant_id",
  clientId: "microsoft_graph_credentials.client_id",
  clientSecret: "microsoft_graph_credentials.client_secret",
};

const getServerFieldErrors = (err: unknown): IFormErrors => {
  const errs: IFormErrors = {};
  CREDENTIAL_FIELDS.forEach((field) => {
    const reason = getErrorReason(err, {
      nameEquals: SERVER_ERROR_NAMES[field],
    });
    if (reason) {
      errs[field] = reason;
    }
  });
  return errs;
};

const MicrosoftGraphPage = () => {
  const { config, isPremiumTier, setConfig } = useContext(AppContext);

  const [formData, setFormData] = useState<IFormData>({
    tenantId: "",
    clientId: "",
    clientSecret: "",
  });
  const [isSaving, setIsSaving] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [formErrors, setFormErrors] = useState<IFormErrors>({});
  // A field is dirty once the admin has typed in it, and stays dirty for the session. Blur only surfaces errors for
  // dirty fields; submit ignores this and validates everything.
  const [dirtyFields, setDirtyFields] = useState<
    Partial<Record<ICredentialField, boolean>>
  >({});

  const { tenantId, clientId, clientSecret } = formData;

  const { data: credentialsResponse, isLoading, isError, refetch } = useQuery<
    IGetMicrosoftGraphCredentialsResponse,
    Error
  >(
    ["microsoft-graph-credentials"],
    () => microsoftGraphCredentialsAPI.getCredentials(),
    {
      enabled: isPremiumTier,
      // Left on React Query's default so the sync status refreshes when the admin returns from the Entra portal. The
      // sync cron runs every 5 minutes, so a snapshot taken at mount goes wrong quickly on a page left open.
      refetchOnWindowFocus: true,
    }
  );

  // Fleet stores at most one credential.
  const storedCredential = credentialsResponse?.microsoft_graph_credentials[0];

  // Seed the form from the stored credential. Leaving the secret field untouched keeps the stored secret.
  useEffect(() => {
    setFormData({
      tenantId: storedCredential?.tenant_id ?? "",
      clientId: storedCredential?.client_id ?? "",
      clientSecret: storedCredential ? STORED_SECRET_PLACEHOLDER : "",
    });
    setFormErrors({});
    setDirtyFields({});
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [storedCredential?.tenant_id, storedCredential?.client_id]);

  const markDirty = (field: ICredentialField) =>
    setDirtyFields((prev) => (prev[field] ? prev : { ...prev, [field]: true }));

  const onInputChange = ({ name, value }: IInputFieldParseTarget) => {
    const field = name as ICredentialField;
    const nextValue = String(value);
    markDirty(field);

    setFormData((prev) => {
      const updated = { ...prev, [field]: nextValue };
      // The stored secret belongs to a specific app registration, so changing either ID invalidates it. Reverting to
      // the stored IDs makes it apply again, so the mask comes back. The dirty check keeps the restore from
      // overwriting a secret field the admin cleared themselves.
      if (storedCredential && field !== "clientSecret") {
        const identityUnchanged = identityMatchesStored(
          updated,
          storedCredential
        );
        if (
          prev.clientSecret === STORED_SECRET_PLACEHOLDER &&
          !identityUnchanged
        ) {
          updated.clientSecret = "";
        } else if (
          prev.clientSecret === "" &&
          identityUnchanged &&
          !dirtyFields.clientSecret
        ) {
          updated.clientSecret = STORED_SECRET_PLACEHOLDER;
        }
      }
      return updated;
    });
  };

  // Re-entry is required whenever the app registration's identity changes, matching the API's own rule.
  const identityChanged =
    !!storedCredential && !identityMatchesStored(formData, storedCredential);

  const secretChanged =
    clientSecret !== "" && clientSecret !== STORED_SECRET_PLACEHOLDER;

  // The secret is only required for a new credential or a changed app registration.
  const secretRequired = !storedCredential || identityChanged;

  // Reverting an edited ID back to its stored value drops the requirement, so the error it produced no longer applies
  // and clears on its own. Matched by message rather than cleared outright, so a server error on the same field stays.
  useEffect(() => {
    if (secretRequired) {
      return;
    }
    setFormErrors((prev) =>
      prev.clientSecret === SECRET_REQUIRED_ERROR
        ? { ...prev, clientSecret: undefined }
        : prev
    );
  }, [secretRequired]);

  // A new credential needs all three values. An existing one can be saved without re-entering the secret, since
  // omitting it keeps the stored one.
  const validate = () => {
    const errs: IFormErrors = {};
    if (tenantId.trim() === "") {
      errs.tenantId = "Enter a tenant ID";
    } else if (!isUUID(tenantId.trim())) {
      errs.tenantId = "Enter a tenant ID in GUID format";
    }
    if (clientId.trim() === "") {
      errs.clientId = "Enter a client ID";
    } else if (!isUUID(clientId.trim())) {
      errs.clientId = "Enter a client ID in GUID format";
    }
    if (secretRequired && !secretChanged) {
      errs.clientSecret = SECRET_REQUIRED_ERROR;
    }
    return errs;
  };

  // Blur re-validates one field and touches no others. A pristine field stays silent, so an admin who tabs through the
  // form without typing never sees an error.
  const onBlurField = (field: ICredentialField) => () => {
    if (!dirtyFields[field]) {
      return;
    }
    const errs = validate();
    setFormErrors((prev) => ({ ...prev, [field]: errs[field] }));
  };

  const onFocusField = (field: ICredentialField) => () =>
    setFormErrors((prev) => ({ ...prev, [field]: undefined }));

  // Saving or deleting always leaves the tenant healthy: a save only succeeds once Fleet has verified the credential
  // against Microsoft Graph, and a delete leaves none that could be invalid.
  const clearInvalidCredentialBanner = () => {
    if (!config?.mdm.microsoft_graph_credential_invalid) {
      return;
    }
    setConfig({
      ...config,
      mdm: { ...config.mdm, microsoft_graph_credential_invalid: false },
    });
  };

  const onSave = async (evt: React.FormEvent) => {
    evt.preventDefault();

    // Guarded here rather than relying on the disabled button alone: a form submits on Enter too, and a double PUT
    // would race two declarative writes against each other.
    if (isSaving) {
      return;
    }

    const errs = validate();
    setFormErrors(errs);
    if (Object.keys(errs).length > 0) {
      return;
    }

    const credential: IMicrosoftGraphCredentialFormData = {
      tenant_id: tenantId.trim(),
      client_id: clientId.trim(),
    };
    // Omitting the secret tells the API to keep the stored one.
    if (secretChanged) {
      credential.client_secret = clientSecret;
    }

    setIsSaving(true);
    try {
      await microsoftGraphCredentialsAPI.applyCredentials([credential]);
      // A secret is stored either way now, so show the mask again.
      setFormData((prev) => ({
        ...prev,
        clientSecret: STORED_SECRET_PLACEHOLDER,
      }));
      notify.success("Successfully saved Microsoft Graph credential.");
      refetch();
      clearInvalidCredentialBanner();
    } catch (e) {
      // Field-specific problems render inline AND toast. A whole-credential failure has no field to attach to, so it is
      // a toast only.
      const fieldErrors = getServerFieldErrors(e);
      const messages = Object.values(fieldErrors);
      if (messages.length > 0) {
        setFormErrors((prev) => ({ ...prev, ...fieldErrors }));
        notify.batch(
          messages.map((message) => ({ variant: "error" as const, message }))
        );
      } else {
        notify.error("Couldn't save Microsoft Graph credential.", {
          response: e,
        });
      }
    } finally {
      setIsSaving(false);
    }
  };

  const onDeleted = () => {
    setShowDeleteModal(false);
    refetch();
    clearInvalidCredentialBanner();
  };

  const renderSyncStatus = () => {
    if (!storedCredential) {
      return null;
    }

    const {
      last_synced_at,
      last_sync_error,
      credential_invalid,
    } = storedCredential;

    // A rejected credential already raises the app-wide banner, so don't duplicate it here.
    const showSyncError = !!last_sync_error && !credential_invalid;

    return (
      <div className={`${baseClass}__sync-status`}>
        <DataSet
          orientation="horizontal"
          textOnly
          title="Last synced"
          value={
            last_synced_at ? (
              <HumanTimeDiffWithDateTip timeString={last_synced_at} />
            ) : (
              "Never"
            )
          }
        />
        {showSyncError && (
          <>
            <TooltipWrapper
              showArrow
              underline={false}
              position="top"
              tipContent={last_sync_error}
            >
              <Icon name="error" />
            </TooltipWrapper>
            <span className="sr-only">{last_sync_error}</span>
          </>
        )}
      </div>
    );
  };

  const renderField = (
    field: ICredentialField,
    label: string,
    extra?: Partial<React.ComponentProps<typeof InputField>>
  ) => (
    <GitOpsModeTooltipWrapper
      isInputField
      renderChildren={(disableChildren) => (
        <InputField
          label={label}
          name={field}
          value={formData[field]}
          onChange={onInputChange}
          parseTarget
          error={formErrors[field]}
          onFocus={onFocusField(field)}
          onBlur={onBlurField(field)}
          inputOptions={{
            maxLength:
              field === "clientSecret" ? SECRET_MAX_LENGTH : ID_MAX_LENGTH,
          }}
          // Fields lock during submission.
          disabled={disableChildren || isSaving}
          {...extra}
        />
      )}
    />
  );

  const renderForm = () => (
    <form onSubmit={onSave}>
      {renderField("tenantId", "Tenant ID")}
      {renderField("clientId", "Client ID")}
      {renderField("clientSecret", "Client secret", {
        type: "password",
        blockAutoComplete: true,
      })}
      <div className={`button-wrap ${baseClass}__form-actions`}>
        <GitOpsModeTooltipWrapper
          renderChildren={(disableChildren) => (
            <Button
              type="submit"
              disabled={disableChildren || isSaving}
              isLoading={isSaving}
            >
              Save
            </Button>
          )}
        />
        {!!storedCredential && (
          <GitOpsModeTooltipWrapper
            renderChildren={(disableChildren) => (
              <Button
                variant="secondary"
                onClick={() => setShowDeleteModal(true)}
                disabled={disableChildren || isSaving}
              >
                Delete
              </Button>
            )}
          />
        )}
      </div>
    </form>
  );

  const renderContent = () => {
    if (!isPremiumTier) {
      return <PremiumFeatureMessage />;
    }
    if (isLoading) {
      return <Spinner />;
    }
    if (isError) {
      return <DataError />;
    }

    return (
      <>
        {renderSyncStatus()}
        {renderForm()}
      </>
    );
  };

  return (
    <MainContent className={baseClass}>
      <>
        <div className={`${baseClass}__header-links`}>
          <BackButton text="Back to MDM" path={PATHS.ADMIN_INTEGRATIONS_MDM} />
        </div>
        <h1>Microsoft Graph</h1>
        <div className={`${baseClass}__content-container`}>
          <PageDescription
            content={
              <>
                Fleet uses a Microsoft Entra app registration to read your
                tenant&apos;s Windows Autopilot devices and show them as pending
                hosts. To create it, follow the instructions in the{" "}
                <CustomLink
                  newTab
                  text="guide"
                  url="https://fleetdm.com/learn-more-about/connect-microsoft-graph"
                />
              </>
            }
          />
          {renderContent()}
        </div>
        {showDeleteModal && (
          <DeleteMicrosoftGraphCredentialModal
            onExit={() => setShowDeleteModal(false)}
            onDeleted={onDeleted}
          />
        )}
      </>
    </MainContent>
  );
};

export default MicrosoftGraphPage;
