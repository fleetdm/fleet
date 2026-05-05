import React, { useCallback, useMemo, useState } from "react";

import { IInputFieldParseTarget } from "interfaces/form_field";
import { HistoricalDataConfigKey } from "interfaces/charts";

import validUrl from "components/forms/validators/valid_url";
import Button from "components/buttons/Button";
import ConfirmDataCollectionDisableModal from "components/ConfirmDataCollectionDisableModal";
import { IConfig } from "interfaces/config";
import { isPremiumTier } from "utilities/permissions/permissions";

import { IAppConfigFormProps } from "../constants";
import HostLifecycleSection from "./components/HostLifecycleSection";
import ActivityDataRetentionSection from "./components/ActivityDataRetentionSection";
import FeaturesSection from "./components/FeaturesSection";
import ServerAuthenticationSection from "./components/ServerAuthenticationSection";

interface IAdvancedConfigFormData {
  ssoUserURL: string;
  mdmAppleServerURL: string;
  domain: string;
  verifySSLCerts: boolean;
  enableStartTLS?: boolean;
  enableHostExpiry: boolean;
  hostExpiryWindow: string;
  deleteActivities: boolean;
  activityExpiryWindow: number;
  disableLiveQuery: boolean;
  disableScripts: boolean;
  disableAIFeatures: boolean;
  disableQueryReports: boolean;
  requireHardwareAttestation: boolean;
  preserveHostActivitiesOnReenrollment: boolean;
  disableHostsActive: boolean;
  disableVulnerabilities: boolean;
}

interface IAdvancedConfigFormErrors {
  ssoUserURL?: string | null;
  mdmAppleServerURL?: string | null;
  domain?: string | null;
  hostExpiryWindow?: string | null;
}

export interface IAdvancedSectionProps {
  isPremiumTier?: boolean;
  onInputChange: AdvancedInputChangeFn;
  onInputBlur?: () => void;
  formData: IAdvancedConfigFormData;
  formErrors?: IAdvancedConfigFormErrors;
  appConfig?: IConfig;
}

const validateFormData = ({
  ssoUserURL,
  mdmAppleServerURL,
  domain,
  hostExpiryWindow,
  enableHostExpiry,
}: IAdvancedConfigFormData) => {
  const errors: Record<string, string> = {};

  if (!ssoUserURL) {
    delete errors.ssoUserURL;
  } else if (
    !validUrl({
      url: ssoUserURL,
    })
  ) {
    errors.ssoUserURL = "SSO user URL is not a valid URL";
  }

  if (!mdmAppleServerURL) {
    delete errors.mdmAppleServerURL;
  } else if (
    !validUrl({
      url: mdmAppleServerURL,
      allowLocalHost: false,
      protocols: ["http", "https"],
    })
  ) {
    errors.mdmAppleServerURL = "Apple MDM server URL is not a valid URL";
  }

  if (!domain) {
    delete errors.domain;
  } else if (!validUrl({ url: domain })) {
    errors.domain = "Domain is not a valid URL";
  }

  if (
    enableHostExpiry &&
    (!hostExpiryWindow || parseInt(hostExpiryWindow, 10) <= 0)
  ) {
    errors.hostExpiryWindow = "Host expiry window must be a positive number";
  }
  return errors;
};

type AdvancedInputChangeFn = ({ name, value }: IInputFieldParseTarget) => void;

const Advanced = ({
  appConfig,
  handleSubmit,
  isUpdatingSettings,
}: IAppConfigFormProps): JSX.Element => {
  const [formData, setFormData] = useState<IAdvancedConfigFormData>({
    ssoUserURL: appConfig.sso_settings?.sso_server_url || "",
    mdmAppleServerURL: appConfig.mdm?.apple_server_url || "",
    domain: appConfig.smtp_settings?.domain || "",
    verifySSLCerts: appConfig.smtp_settings?.verify_ssl_certs || false,
    enableStartTLS: appConfig.smtp_settings?.enable_start_tls,
    enableHostExpiry:
      appConfig.host_expiry_settings.host_expiry_enabled || false,
    hostExpiryWindow:
      (appConfig.host_expiry_settings.host_expiry_window &&
        appConfig.host_expiry_settings.host_expiry_window.toString()) ||
      "0",
    deleteActivities:
      appConfig.activity_expiry_settings.activity_expiry_enabled || false,
    activityExpiryWindow:
      appConfig.activity_expiry_settings.activity_expiry_window || 30,
    disableLiveQuery: appConfig.server_settings.live_query_disabled || false,
    disableScripts: appConfig.server_settings.scripts_disabled || false,
    disableAIFeatures: appConfig.server_settings.ai_features_disabled || false,
    disableQueryReports:
      appConfig.server_settings.query_reports_disabled || false,
    requireHardwareAttestation:
      appConfig.mdm?.apple_require_hardware_attestation || false,
    preserveHostActivitiesOnReenrollment:
      appConfig.activity_expiry_settings
        .preserve_host_activities_on_reenrollment || false,
    disableHostsActive: !appConfig.features.historical_data.uptime,
    disableVulnerabilities: !appConfig.features.historical_data.vulnerabilities,
  });

  const [formErrors, setFormErrors] = useState<IAdvancedConfigFormErrors>({});
  const [confirmModalOpen, setConfirmModalOpen] = useState(false);

  const onInputChange: AdvancedInputChangeFn = ({
    name,
    value,
  }: IInputFieldParseTarget) => {
    const newFormData = { ...formData, [name]: value };
    setFormData(newFormData);
    const newErrs = validateFormData(newFormData);
    // only set errors that are updates of existing errors
    // new errors are only set onBlur
    const errsToSet: Record<string, string> = {};
    Object.keys(formErrors).forEach((k) => {
      if (newErrs[k]) {
        errsToSet[k] = newErrs[k];
      }
    });
    setFormErrors(errsToSet);
  };

  const onInputBlur = () => {
    setFormErrors(validateFormData(formData));
  };

  const buildPayload = useCallback(
    () => ({
      server_settings: {
        live_reporting_disabled: formData.disableLiveQuery,
        discard_reports_data: formData.disableQueryReports,
        scripts_disabled: formData.disableScripts,
        deferred_save_host: appConfig.server_settings.deferred_save_host,
        ai_features_disabled: formData.disableAIFeatures,
      },
      smtp_settings: {
        domain: formData.domain,
        verify_ssl_certs: formData.verifySSLCerts,
        enable_start_tls: formData.enableStartTLS || false,
      },
      host_expiry_settings: {
        host_expiry_enabled: formData.enableHostExpiry,
        host_expiry_window:
          parseInt(formData.hostExpiryWindow, 10) || undefined,
      },
      activity_expiry_settings: {
        activity_expiry_enabled: formData.deleteActivities,
        activity_expiry_window: formData.activityExpiryWindow || undefined,
        preserve_host_activities_on_reenrollment:
          formData.preserveHostActivitiesOnReenrollment,
      },
      mdm: {
        apple_server_url: formData.mdmAppleServerURL,
        apple_require_hardware_attestation: formData.requireHardwareAttestation,
      },
      sso_settings: {
        sso_server_url: formData.ssoUserURL,
      },
      features: {
        historical_data: {
          uptime: !formData.disableHostsActive,
          vulnerabilities: !formData.disableVulnerabilities,
        },
      },
    }),
    [formData, appConfig.server_settings.deferred_save_host]
  );

  const datasetsBeingDisabled = useMemo<HistoricalDataConfigKey[]>(() => {
    const list: HistoricalDataConfigKey[] = [];
    const original = appConfig.features.historical_data;
    if (original.uptime && formData.disableHostsActive) {
      list.push("uptime");
    }
    if (original.vulnerabilities && formData.disableVulnerabilities) {
      list.push("vulnerabilities");
    }
    return list;
  }, [appConfig, formData.disableHostsActive, formData.disableVulnerabilities]);

  const performSave = useCallback(async () => {
    const ok = await handleSubmit(buildPayload());
    if (ok) {
      setConfirmModalOpen(false);
    }
  }, [handleSubmit, buildPayload]);

  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    const errs = validateFormData(formData);
    if (Object.keys(errs).length > 0) {
      setFormErrors(errs);
      return;
    }

    if (datasetsBeingDisabled.length > 0) {
      setConfirmModalOpen(true);
      return;
    }

    performSave();
  };

  const isPremiumLicense = isPremiumTier(appConfig);

  return (
    <div>
      <form onSubmit={onFormSubmit} autoComplete="off">
        <HostLifecycleSection
          onInputChange={onInputChange}
          formData={formData}
          formErrors={formErrors}
          isPremiumTier={isPremiumLicense}
        />
        <ActivityDataRetentionSection
          formData={formData}
          onInputChange={onInputChange}
        />
        <FeaturesSection formData={formData} onInputChange={onInputChange} />
        <ServerAuthenticationSection
          formData={formData}
          onInputChange={onInputChange}
          onInputBlur={onInputBlur}
          formErrors={formErrors}
          appConfig={appConfig}
        />
        <Button
          type="submit"
          disabled={Object.keys(formErrors).length > 0}
          className="save-loading button-wrap"
          isLoading={isUpdatingSettings}
        >
          Save
        </Button>
      </form>
      {confirmModalOpen && (
        <ConfirmDataCollectionDisableModal
          scope="global"
          datasets={datasetsBeingDisabled}
          isUpdating={isUpdatingSettings}
          onConfirm={performSave}
          onCancel={() => setConfirmModalOpen(false)}
        />
      )}
    </div>
  );
};

export default Advanced;
