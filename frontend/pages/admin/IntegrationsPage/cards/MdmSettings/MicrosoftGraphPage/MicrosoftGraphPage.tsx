import React, { useContext, useEffect, useState } from "react";
import { useQuery } from "react-query";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { UNCHANGED_PASSWORD_API_RESPONSE } from "utilities/constants";
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
import MainContent from "components/MainContent";
import PageDescription from "components/PageDescription";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import Spinner from "components/Spinner";
import { notify } from "components/ToastNotification";

import DeleteMicrosoftGraphCredentialModal from "./DeleteMicrosoftGraphCredentialModal";

const baseClass = "microsoft-graph-page";

// Entra IDs and secrets are stored in varchar(255) columns.
const FIELD_MAX_LENGTH = 255;

interface IFormErrors {
  tenantId?: string;
  clientId?: string;
  clientSecret?: string;
}

const MicrosoftGraphPage = () => {
  const { isPremiumTier, setConfig } = useContext(AppContext);

  const [tenantId, setTenantId] = useState("");
  const [clientId, setClientId] = useState("");
  const [clientSecret, setClientSecret] = useState("");
  const [isSaving, setIsSaving] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [formErrors, setFormErrors] = useState<IFormErrors>({});

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
  useEffect(() => {
    setTenantId(storedCredential?.tenant_id ?? "");
    setClientId(storedCredential?.client_id ?? "");
    setClientSecret(storedCredential ? UNCHANGED_PASSWORD_API_RESPONSE : "");
    setFormErrors({});
  }, [storedCredential]);

  // The stored secret belongs to a specific app registration, so changing either ID invalidates it. Clearing the mask
  // forces re-entry, which is the same pattern AccountProvisioning and the certificate-authority edit flow use, and it
  // surfaces inline what the API would otherwise reject with a 422.
  const clearMaskedSecretOnIdChange = () => {
    setClientSecret((prev) =>
      prev === UNCHANGED_PASSWORD_API_RESPONSE ? "" : prev
    );
  };

  const onChangeTenantId = (value: string) => {
    setTenantId(value);
    if (storedCredential && value.trim() !== storedCredential.tenant_id) {
      clearMaskedSecretOnIdChange();
    }
  };

  const onChangeClientId = (value: string) => {
    setClientId(value);
    if (storedCredential && value.trim() !== storedCredential.client_id) {
      clearMaskedSecretOnIdChange();
    }
  };

  // Re-entry is required whenever the app registration's identity changes, matching the API's own rule.
  const identityChanged =
    !!storedCredential &&
    (tenantId.trim() !== storedCredential.tenant_id ||
      clientId.trim() !== storedCredential.client_id);

  const secretChanged =
    clientSecret !== "" && clientSecret !== UNCHANGED_PASSWORD_API_RESPONSE;

  // A new credential needs all three values. An existing one can be saved without re-entering the secret, since
  // omitting it keeps the stored one.
  const validate = () => {
    const errs: IFormErrors = {};
    if (tenantId.trim() === "") {
      errs.tenantId = "Enter a tenant ID";
    }
    if (clientId.trim() === "") {
      errs.clientId = "Enter a client ID";
    }
    if ((!storedCredential || identityChanged) && !secretChanged) {
      errs.clientSecret = "Enter a client secret";
    }
    return errs;
  };

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
      notify.success("Successfully saved Microsoft Graph credential.");
      refetch();
      await refreshAppConfig();
    } catch (e) {
      notify.error(
        getErrorReason(e) || "Couldn't save Microsoft Graph credential."
      );
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

  const renderForm = () => (
    <form className={`${baseClass}__form`} onSubmit={onSave}>
      <GitOpsModeTooltipWrapper
        isInputField
        renderChildren={(disableChildren) => (
          <InputField
            label="Tenant ID"
            name="tenantId"
            value={tenantId}
            onChange={onChangeTenantId}
            error={formErrors.tenantId}
            onFocus={() =>
              setFormErrors((prev) => ({ ...prev, tenantId: undefined }))
            }
            inputOptions={{ maxLength: FIELD_MAX_LENGTH }}
            disabled={disableChildren}
          />
        )}
      />
      <GitOpsModeTooltipWrapper
        isInputField
        renderChildren={(disableChildren) => (
          <InputField
            label="Client ID"
            name="clientId"
            value={clientId}
            onChange={onChangeClientId}
            error={formErrors.clientId}
            onFocus={() =>
              setFormErrors((prev) => ({ ...prev, clientId: undefined }))
            }
            inputOptions={{ maxLength: FIELD_MAX_LENGTH }}
            disabled={disableChildren}
          />
        )}
      />
      <GitOpsModeTooltipWrapper
        isInputField
        renderChildren={(disableChildren) => (
          <InputField
            label="Client secret"
            name="clientSecret"
            type="password"
            value={clientSecret}
            onChange={setClientSecret}
            error={formErrors.clientSecret}
            onFocus={() =>
              setFormErrors((prev) => ({ ...prev, clientSecret: undefined }))
            }
            inputOptions={{ maxLength: FIELD_MAX_LENGTH }}
            disabled={disableChildren}
          />
        )}
      />
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
                  url="https://fleetdm.com/learn-more-about/connect-microsoft-entra"
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
