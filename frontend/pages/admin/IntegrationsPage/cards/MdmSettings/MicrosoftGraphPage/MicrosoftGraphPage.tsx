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

import BackButton from "components/BackButton";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import DataError from "components/DataError";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
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
  const { isPremiumTier } = useContext(AppContext);

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
      refetchOnWindowFocus: false,
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
    if (!storedCredential && !secretChanged) {
      errs.clientSecret = "Enter a client secret";
    }
    return errs;
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
  };

  const renderSyncStatus = () => {
    if (!storedCredential) {
      return null;
    }

    const { last_synced_at, last_sync_error } = storedCredential;

    return (
      <div className={`${baseClass}__sync-status`}>
        <div className={`${baseClass}__sync-status-row`}>
          <span className={`${baseClass}__sync-status-label`}>Last synced</span>
          {last_synced_at ? (
            <HumanTimeDiffWithDateTip timeString={last_synced_at} />
          ) : (
            <span>Never</span>
          )}
        </div>
        {!!last_sync_error && (
          <div className={`${baseClass}__sync-error`}>{last_sync_error}</div>
        )}
      </div>
    );
  };

  const renderForm = () => (
    <form className={`${baseClass}__form`} onSubmit={onSave}>
      <GitOpsModeTooltipWrapper
        tipOffset={8}
        renderChildren={(disableChildren) => (
          <>
            <InputField
              label="Tenant ID"
              name="tenantId"
              value={tenantId}
              onChange={setTenantId}
              error={formErrors.tenantId}
              onFocus={() =>
                setFormErrors((prev) => ({ ...prev, tenantId: undefined }))
              }
              inputOptions={{ maxLength: FIELD_MAX_LENGTH }}
              disabled={disableChildren}
              tooltip="The directory (tenant) ID of the Microsoft Entra tenant whose Autopilot devices Fleet syncs."
            />
            <InputField
              label="Client ID"
              name="clientId"
              value={clientId}
              onChange={setClientId}
              error={formErrors.clientId}
              onFocus={() =>
                setFormErrors((prev) => ({ ...prev, clientId: undefined }))
              }
              inputOptions={{ maxLength: FIELD_MAX_LENGTH }}
              disabled={disableChildren}
              tooltip="The application (client) ID of the Entra app registration Fleet authenticates as."
            />
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
              tooltip="The client secret for the app registration. Fleet never displays a stored secret."
            />
          </>
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
                hosts. To create the app registration, follow the instructions
                in the{" "}
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
