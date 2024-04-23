import React, { useState, useEffect } from "react";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import SectionHeader from "components/SectionHeader";

import {
  IAppConfigFormProps,
  IFormField,
  IAppConfigFormErrors,
} from "../constants";

const baseClass = "app-config-form";

const Advanced = ({
  appConfig,
  handleSubmit,
  isUpdatingSettings,
}: IAppConfigFormProps): JSX.Element => {
  const [formData, setFormData] = useState({
    domain: appConfig.smtp_settings?.domain || "",
    verifySSLCerts: appConfig.smtp_settings?.verify_ssl_certs || false,
    enableStartTLS: appConfig.smtp_settings?.enable_start_tls,
    enableHostExpiry:
      appConfig.host_expiry_settings.host_expiry_enabled || false,
    hostExpiryWindow: appConfig.host_expiry_settings.host_expiry_window || 0,
    disableLiveQuery: appConfig.server_settings.live_query_disabled || false,
    disableQueryReports:
      appConfig.server_settings.query_reports_disabled || false,
    disableScripts: appConfig.server_settings.scripts_disabled || false,
  });

  const {
    domain,
    verifySSLCerts,
    enableStartTLS,
    enableHostExpiry,
    hostExpiryWindow,
    disableLiveQuery,
    disableScripts,
    disableQueryReports,
  } = formData;

  const [formErrors, setFormErrors] = useState<IAppConfigFormErrors>({});

  const handleInputChange = ({ name, value }: IFormField) => {
    setFormData({ ...formData, [name]: value });
  };

  useEffect(() => {
    // validate desired form fields
    const errors: IAppConfigFormErrors = {};

    if (enableHostExpiry && (!hostExpiryWindow || hostExpiryWindow <= 0)) {
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
      },
      smtp_settings: {
        domain,
        verify_ssl_certs: verifySSLCerts,
        enable_start_tls: enableStartTLS,
      },
      host_expiry_settings: {
        host_expiry_enabled: enableHostExpiry,
        host_expiry_window: Number(hostExpiryWindow),
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
            onChange={handleInputChange}
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
            onChange={handleInputChange}
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
            onChange={handleInputChange}
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
            onChange={handleInputChange}
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
              onChange={handleInputChange}
              name="hostExpiryWindow"
              value={hostExpiryWindow}
              parseTarget
              error={formErrors.host_expiry_window}
            />
          )}
          <Checkbox
            onChange={handleInputChange}
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
            onChange={handleInputChange}
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
            onChange={handleInputChange}
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
