import React, { useState, useMemo } from "react";

import validUrl from "components/forms/validators/valid_url";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import SectionHeader from "components/SectionHeader";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";

import { ACTIVITY_EXPIRY_WINDOW_DROPDOWN_OPTIONS } from "utilities/constants";
import { getCustomDropdownOptions } from "utilities/helpers";

import { IAppConfigFormProps, IFormField } from "../constants";

const baseClass = "app-config-form";

interface IAdvancedConfigFormData {
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
}

interface IAdvancedConfigFormErrors {
  mdmAppleServerURL?: string | null;
  domain?: string | null;
  hostExpiryWindow?: string | null;
}

const validateFormData = ({
  mdmAppleServerURL,
  domain,
  hostExpiryWindow,
  enableHostExpiry,
}: IAdvancedConfigFormData) => {
  const errors: Record<string, string> = {};

  if (!mdmAppleServerURL) {
    delete errors.mdmAppleServerURL;
  } else if (!validUrl({ url: mdmAppleServerURL })) {
    errors.mdmAppleServerURL = `${mdmAppleServerURL} is not a valid URL`;
  }

  if (!domain) {
    delete errors.domain;
  } else if (!validUrl({ url: domain })) {
    errors.domain = `${domain} is not a valid URL`;
  }

  if (
    enableHostExpiry &&
    (!hostExpiryWindow || parseInt(hostExpiryWindow, 10) <= 0)
  ) {
    errors.hostExpiryWindow = "Host expiry window must be a positive number";
  }
  return errors;
};

const Advanced = ({
  appConfig,
  handleSubmit,
  isUpdatingSettings,
}: IAppConfigFormProps): JSX.Element => {
  const [formData, setFormData] = useState<IAdvancedConfigFormData>({
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
      appConfig.activity_expiry_settings?.activity_expiry_enabled || false,
    activityExpiryWindow:
      appConfig.activity_expiry_settings?.activity_expiry_window || 30,
    disableLiveQuery: appConfig.server_settings.live_query_disabled || false,
    disableScripts: appConfig.server_settings.scripts_disabled || false,
    disableAIFeatures: appConfig.server_settings.ai_features_disabled || false,
    disableQueryReports:
      appConfig.server_settings.query_reports_disabled || false,
  });

  const {
    mdmAppleServerURL,
    domain,
    verifySSLCerts,
    enableStartTLS,
    enableHostExpiry,
    hostExpiryWindow,
    deleteActivities,
    activityExpiryWindow,
    disableLiveQuery,
    disableScripts,
    disableAIFeatures,
    disableQueryReports,
  } = formData;

  const [formErrors, setFormErrors] = useState<IAdvancedConfigFormErrors>({});

  const activityExpiryWindowOptions = useMemo(
    () =>
      getCustomDropdownOptions(
        ACTIVITY_EXPIRY_WINDOW_DROPDOWN_OPTIONS,
        activityExpiryWindow,
        // it's safe to assume that frequency is a number
        (frequency: number | string) => `${frequency as number} days`
      ),
    // intentionally leave activityExpiryWindow out of the dependencies, so that the custom
    // options are maintained even if the user changes the frequency in the UI
    [deleteActivities]
  );

  const onInputChange = ({ name, value }: IFormField) => {
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

  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    const errs = validateFormData(formData);
    if (Object.keys(errs).length > 0) {
      setFormErrors(errs);
      return;
    }

    // Formatting of API not UI
    const formDataToSubmit = {
      server_settings: {
        live_query_disabled: disableLiveQuery,
        query_reports_disabled: disableQueryReports,
        scripts_disabled: disableScripts,
        deferred_save_host: appConfig.server_settings.deferred_save_host,
        ai_features_disabled: disableAIFeatures,
      },
      smtp_settings: {
        domain,
        verify_ssl_certs: verifySSLCerts,
        enable_start_tls: enableStartTLS || false,
      },
      host_expiry_settings: {
        host_expiry_enabled: enableHostExpiry,
        host_expiry_window: parseInt(hostExpiryWindow, 10) || undefined,
      },
      activity_expiry_settings: {
        activity_expiry_enabled: deleteActivities,
        activity_expiry_window: activityExpiryWindow || undefined,
      },
      mdm: {
        apple_server_url: mdmAppleServerURL,
      },
    };

    handleSubmit(formDataToSubmit);
  };

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__section`}>
        <SectionHeader title="Advanced options" />
        <form onSubmit={onFormSubmit} autoComplete="off">
          <p className={`${baseClass}__section-description`}>
            Most users do not need to modify these options.
          </p>
          {appConfig.mdm.enabled_and_configured && (
            <InputField
              label="Apple MDM server URL"
              onChange={onInputChange}
              onBlur={onInputBlur}
              name="mdmAppleServerURL"
              value={mdmAppleServerURL}
              parseTarget
              error={formErrors.mdmAppleServerURL}
              tooltip="Update this URL if you're self-hosting Fleet and you want your hosts to talk to this URL for MDM features. If not configured, hosts will use the base URL of the Fleet instance."
              helpText="If this URL changes and hosts already have MDM turned on, the end users will have to turn MDM off and back on to use MDM features."
            />
          )}
          <InputField
            label="Domain"
            onChange={onInputChange}
            onBlur={onInputBlur}
            name="domain"
            value={domain}
            parseTarget
            error={formErrors.domain}
            tooltip={
              <>
                If you need to specify a HELO domain, <br />
                you can do it here{" "}
                <em>
                  (Default: <strong>Blank</strong>)
                </em>
              </>
            }
          />
          <Checkbox
            onChange={onInputChange}
            name="verifySSLCerts"
            value={verifySSLCerts}
            parseTarget
            tooltipContent={
              <>
                Turn this off (not recommended) <br />
                if you use a self-signed certificate{" "}
                <em>
                  <br />
                  (Default: <strong>On</strong>)
                </em>
              </>
            }
          >
            Verify SSL certs
          </Checkbox>
          <Checkbox
            onChange={onInputChange}
            name="enableStartTLS"
            value={enableStartTLS}
            parseTarget
            tooltipContent={
              <>
                Detects if STARTTLS is enabled <br />
                in your SMTP server and starts <br />
                to use it.{" "}
                <em>
                  (Default: <strong>On</strong>)
                </em>
              </>
            }
          >
            Enable STARTTLS
          </Checkbox>
          <Checkbox
            onChange={onInputChange}
            name="enableHostExpiry"
            value={enableHostExpiry}
            parseTarget
            tooltipContent={
              <>
                When enabled, allows automatic cleanup of
                <br />
                hosts that have not communicated with Fleet in
                <br />
                the number of days specified in the{" "}
                <strong>
                  Host expiry
                  <br />
                  window
                </strong>{" "}
                setting.{" "}
                <em>
                  (Default: <strong>Off</strong>)
                </em>
              </>
            }
          >
            Host expiry
          </Checkbox>
          {enableHostExpiry && (
            <InputField
              label="Host expiry window"
              type="number"
              onChange={onInputChange}
              name="hostExpiryWindow"
              value={hostExpiryWindow}
              parseTarget
              error={formErrors.hostExpiryWindow}
            />
          )}
          <Checkbox
            onChange={onInputChange}
            name="deleteActivities"
            value={deleteActivities}
            parseTarget
            tooltipContent={
              <>
                When enabled, allows automatic cleanup of audit logs older than
                the number of days specified in the{" "}
                <em>Audit log retention window</em> setting.
                <em>
                  (Default: <strong>Off</strong>)
                </em>
              </>
            }
          >
            Delete activities
          </Checkbox>
          {deleteActivities && (
            <Dropdown
              searchable={false}
              options={activityExpiryWindowOptions}
              onChange={onInputChange}
              placeholder="Select"
              value={activityExpiryWindow}
              label="Max activity age"
              name="activityExpiryWindow"
              parseTarget
            />
          )}
          <Checkbox
            onChange={onInputChange}
            name="disableLiveQuery"
            value={disableLiveQuery}
            parseTarget
            tooltipContent={
              <>
                When enabled, disables the ability to run live queries <br />
                (ad hoc queries executed via the UI or fleetctl).{" "}
                <em>
                  (Default: <strong>Off</strong>)
                </em>
              </>
            }
          >
            Disable live queries
          </Checkbox>
          <Checkbox
            onChange={onInputChange}
            name="disableScripts"
            value={disableScripts}
            parseTarget
            tooltipContent={
              <>
                Disabling script execution will block access to run scripts.
                <br />
                Scripts may still be added and removed in the UI and API.
                <br />
                <em>
                  (Default: <b>Off</b>)
                </em>
              </>
            }
            helpText="Features that run scripts under-the-hood (e.g. software install, lock/wipe) will still be available."
          >
            Disable script execution features
          </Checkbox>
          <Checkbox
            onChange={onInputChange}
            name="disableAIFeatures"
            value={disableAIFeatures}
            parseTarget
            tooltipContent={
              <>
                When enabled, disables AI features such as pre-filling forms
                <br />
                with descriptions generated by a large language model
                <br />
                (LLM).{" "}
                <em>
                  (Default: <strong>Off</strong>)
                </em>
              </>
            }
            helpText="If enabled, only policy queries (SQL) are sent to the LLM. Fleet doesn’t use this data to train models."
          >
            Disable generative AI features
          </Checkbox>
          <Checkbox
            onChange={onInputChange}
            name="disableQueryReports"
            value={disableQueryReports}
            parseTarget
            tooltipContent={
              <>
                <>
                  Disabling query reports will decrease database usage, <br />
                  but will prevent you from accessing query results in
                  <br />
                  Fleet and will delete existing reports. This can also be{" "}
                  <br />
                  disabled on a per-query basis by enabling &quot;Discard <br />
                  data&quot;.{" "}
                  <em>
                    (Default: <b>Off</b>)
                  </em>
                </>
              </>
            }
            helpText="Enabling this setting will delete all existing query reports in Fleet."
          >
            Disable query reports
          </Checkbox>
          <Button
            type="submit"
            variant="brand"
            disabled={Object.keys(formErrors).length > 0}
            className="save-loading button-wrap"
            isLoading={isUpdatingSettings}
          >
            Save
          </Button>
        </form>
      </div>
    </div>
  );
};

export default Advanced;
