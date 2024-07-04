import React, { useState, useEffect, useMemo } from "react";

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
  host_expiry_window?: string | null;
}

const Advanced = ({
  appConfig,
  handleSubmit,
  isUpdatingSettings,
}: IAppConfigFormProps): JSX.Element => {
  const [formData, setFormData] = useState<IAdvancedConfigFormData>({
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
    setFormData({ ...formData, [name]: value });
  };

  useEffect(() => {
    // validate desired form fields
    const errors: IAdvancedConfigFormErrors = {};

    if (
      enableHostExpiry &&
      (!hostExpiryWindow || parseInt(hostExpiryWindow, 10) <= 0)
    ) {
      errors.host_expiry_window =
        "Host expiry window must be a positive number";
    }

    setFormErrors(errors);
  }, [enableHostExpiry, hostExpiryWindow]);

  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

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
          <InputField
            label="Domain"
            onChange={onInputChange}
            name="domain"
            value={domain}
            parseTarget
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
              error={formErrors.host_expiry_window}
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
                Disabling scripts will block access to run scripts. Scripts{" "}
                <br /> may still be added and removed in the UI and API. <br />
                <em>
                  (Default: <strong>Off</strong>)
                </em>
              </>
            }
          >
            Disable scripts
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
