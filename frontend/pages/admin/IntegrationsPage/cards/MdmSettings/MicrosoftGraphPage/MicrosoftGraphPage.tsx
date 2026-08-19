import React, { useContext, useEffect, useState } from "react";
import { useQuery } from "react-query";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { UNCHANGED_PASSWORD_API_RESPONSE } from "utilities/constants";
import { IInputFieldParseTarget } from "interfaces/form_field";
import { IMicrosoftGraphCredentialFormData } from "interfaces/microsoft_graph_credential";
import microsoftGraphCredentialsAPI, {
  IGetMicrosoftGraphCredentialsResponse,
} from "services/entities/microsoft_graph_credentials";
import { getErrorReason } from "interfaces/errors";
import configAPI from "services/entities/config";

import BackButton from "components/BackButton";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import DataError from "components/DataError";
import DataSet from "components/DataSet";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import IconStatusMessage from "components/IconStatusMessage";
import { HumanTimeDiffWithDateTip } from "components/HumanTimeDiffWithDateTip";
import InputField from "components/forms/fields/InputField";
import isUUID from "components/forms/validators/valid_uuid";
import MainContent from "components/MainContent";
import PageDescription from "components/PageDescription";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import Spinner from "components/Spinner";
import { notify } from "components/ToastNotification";

import DeleteMicrosoftGraphCredentialModal from "./DeleteMicrosoftGraphCredentialModal";

const baseClass = "microsoft-graph-page";

// Entra IDs and secrets are stored in varchar(255) columns.
const FIELD_MAX_LENGTH = 255;

// The API lower-cases both IDs before comparing, so the UI must too or re-pasting the same app registration in a
// different case looks like an identity change and needlessly demands a fresh secret.
const sameEntraID = (a: string, b: string) =>
  a.toLowerCase() === b.toLowerCase();

type ICredentialField = "tenantId" | "clientId" | "clientSecret";

type IFormData = Record<ICredentialField, string>;

type IFormErrors = Partial<Record<ICredentialField, string>>;

const SECRET_REQUIRED_ERROR = "Enter a client secret";

const CREDENTIAL_FIELDS: ICredentialField[] = [
  "tenantId",
  "clientId",
  "clientSecret",
];

// The API reports per-field problems under these names. Anything it reports under the bare `microsoft_graph_credentials`
// key is a whole-credential failure (verification, licensing, missing private key) with no field to attach it to.
// Lets one change handler compare either ID against the credential Fleet already stores.
const STORED_ID_KEYS = {
  tenantId: "tenant_id",
  clientId: "client_id",
} as const;

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
  const { isPremiumTier, setConfig } = useContext(AppContext);

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
  const storedCredential =
    credentialsResponse?.microsoft_graph_credentials?.[0];

  // Seed the form from the stored credential. The API never returns the secret, so the field shows the masked
  // placeholder to signal that one is stored; leaving it untouched keeps the stored secret.
  // Keyed on identity, not on the credential object. A background refetch (focus, or after the 5-minute sync rewrites
  // last_synced_at) yields a new object, and re-seeding on that would discard whatever the admin is mid-way through
  // typing. onSave restores the mask itself, so nothing here needs to react to sync state.
  useEffect(() => {
    setFormData({
      tenantId: storedCredential?.tenant_id ?? "",
      clientId: storedCredential?.client_id ?? "",
      clientSecret: storedCredential ? UNCHANGED_PASSWORD_API_RESPONSE : "",
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
      // The stored secret belongs to a specific app registration, so changing either ID invalidates it. Clearing the
      // mask forces re-entry, which is the same pattern AccountProvisioning and the certificate-authority edit flow
      // use, and it surfaces inline what the API would otherwise reject with a 422.
      if (
        storedCredential &&
        field !== "clientSecret" &&
        prev.clientSecret === UNCHANGED_PASSWORD_API_RESPONSE &&
        !sameEntraID(nextValue.trim(), storedCredential[STORED_ID_KEYS[field]])
      ) {
        updated.clientSecret = "";
      }
      return updated;
    });
  };

  // Re-entry is required whenever the app registration's identity changes, matching the API's own rule.
  const identityChanged =
    !!storedCredential &&
    (!sameEntraID(tenantId.trim(), storedCredential.tenant_id) ||
      !sameEntraID(clientId.trim(), storedCredential.client_id));

  const secretChanged =
    clientSecret !== "" && clientSecret !== UNCHANGED_PASSWORD_API_RESPONSE;

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

  // The app-wide banner reads config.mdm.microsoft_graph_credential_invalid from AppContext, and App only refreshes
  // that on a pathname change. Saving or deleting recomputes the aggregate server-side, so pull the config again or a
  // repaired credential leaves the banner up until the admin navigates away.
  const refreshAppConfig = async () => {
    try {
      setConfig(await configAPI.loadAll());
    } catch {
      // Non-fatal: the banner clears on the next navigation.
    }
  };

  const onSave = async (evt: React.FormEvent) => {
    evt.preventDefault();

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
      // A secret is stored either way now, so show the mask again. The seeding effect only fires when the identity
      // changed, so a save that kept the same tenant and client would otherwise leave the raw secret on screen.
      setFormData((prev) => ({
        ...prev,
        clientSecret: UNCHANGED_PASSWORD_API_RESPONSE,
      }));
      notify.success("Successfully saved Microsoft Graph credential.");
      refetch();
      await refreshAppConfig();
    } catch (e) {
      // Field-specific problems render inline AND toast: the form can be scrolled past the errored field after submit.
      // A whole-credential failure has no field to attach to, so it is a toast only.
      const fieldErrors = getServerFieldErrors(e);
      const messages = Object.values(fieldErrors);
      if (messages.length > 0) {
        setFormErrors((prev) => ({ ...prev, ...fieldErrors }));
        notify.batch(
          messages.map((message) => ({ variant: "error" as const, message }))
        );
      } else {
        notify.error(
          getErrorReason(e) || "Couldn't save Microsoft Graph credential."
        );
      }
    } finally {
      setIsSaving(false);
    }
  };

  const onDeleted = () => {
    setShowDeleteModal(false);
    refetch();
    refreshAppConfig();
  };

  const renderSyncStatus = () => {
    if (!storedCredential) {
      return null;
    }

    const { last_synced_at, last_sync_error } = storedCredential;

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
        {!!last_sync_error && (
          <IconStatusMessage
            className={`${baseClass}__sync-error`}
            iconName="error"
            message={last_sync_error}
          />
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
          inputOptions={{ maxLength: FIELD_MAX_LENGTH }}
          disabled={disableChildren}
          {...extra}
        />
      )}
    />
  );

  const renderForm = () => (
    <form className={`${baseClass}__form`} onSubmit={onSave}>
      {renderField("tenantId", "Tenant ID")}
      {renderField("clientId", "Client ID")}
      {/* blockAutoComplete renders autocomplete="new-password" so a password manager neither offers to fill this nor
          prompts to save it. This is a service principal's secret, not a user login, and the field carries a mask
          rather than the real value. InputField's ignore1password default already emits data-1p-ignore. */}
      {renderField("clientSecret", "Client secret", {
        type: "password",
        blockAutoComplete: true,
      })}
      <div className={`${baseClass}__form-actions`}>
        <GitOpsModeTooltipWrapper
          renderChildren={(disableChildren) => (
            <Button
              type="submit"
              disabled={disableChildren}
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
                disabled={disableChildren}
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
                hosts. The app registration needs the{" "}
                <b>DeviceManagementServiceConfig.Read.All</b> application
                permission, with admin consent granted. To create it, follow the
                instructions in the{" "}
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
