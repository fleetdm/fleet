import React, { useState, useEffect } from "react";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
// @ts-ignore
import InputField from "components/forms/fields/InputField";

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
  const [formData, setFormData] = useState<any>({
    domain: appConfig.smtp_settings.domain || "",
    verifySSLCerts: appConfig.smtp_settings.verify_ssl_certs || false,
    enableStartTLS: appConfig.smtp_settings.enable_start_tls,
    enableHostExpiry:
      appConfig.host_expiry_settings.host_expiry_enabled || false,
    hostExpiryWindow: appConfig.host_expiry_settings.host_expiry_window || 0,
    disableLiveQuery: appConfig.server_settings.live_query_disabled || false,
  });

  const {
    domain,
    verifySSLCerts,
    enableStartTLS,
    enableHostExpiry,
    hostExpiryWindow,
    disableLiveQuery,
  } = formData;

  const [formErrors, setFormErrors] = useState<IAppConfigFormErrors>({});

  const handleInputChange = ({ name, value }: IFormField) => {
    setFormData({ ...formData, [name]: value });
  };

  const validateForm = () => {
    const errors: IAppConfigFormErrors = {};

    if (enableHostExpiry) {
      if (!hostExpiryWindow || hostExpiryWindow <= 0) {
        errors.host_expiry_window =
          "Host expiry window must be a positive number";
      }
    }

    setFormErrors(errors);
  };

  useEffect(() => {
    validateForm();
  }, [enableHostExpiry]);

  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    // Formatting of API not UI
    const formDataToSubmit = {
      server_settings: {
        server_url: appConfig.server_settings.server_url || "",
        live_query_disabled: disableLiveQuery,
        enable_analytics: appConfig.server_settings.enable_analytics,
      },
      smtp_settings: {
        enable_smtp: appConfig.smtp_settings.enable_smtp || false,
        sender_address: appConfig.smtp_settings.sender_address || "",
        server: appConfig.smtp_settings.server || "",
        port: Number(appConfig.smtp_settings.port),
        authentication_type: appConfig.smtp_settings.authentication_type || "",
        user_name: appConfig.smtp_settings.user_name || "",
        password: appConfig.smtp_settings.password || "",
        enable_ssl_tls: appConfig.smtp_settings.enable_ssl_tls || false,
        authentication_method:
          appConfig.smtp_settings.authentication_method || "",
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
    <form className={baseClass} onSubmit={onFormSubmit} autoComplete="off">
      <div className={`${baseClass}__section`}>
        <h2>Advanced options</h2>
        <div className={`${baseClass}__advanced-options`}>
          <p className={`${baseClass}__section-description`}>
            Most users do not need to modify these options.
          </p>
          <div className={`${baseClass}__inputs`}>
            <div className={`${baseClass}__form-fields`}>
              <InputField
                label="Domain"
                onChange={handleInputChange}
                name="domain"
                value={domain}
                parseTarget
                tooltip={
                  '<p>If you need to specify a HELO domain, <br />you can do it here <em className="hint hint--brand">(Default: <strong>Blank</strong>)</em></p>'
                }
              />
              <Checkbox
                onChange={handleInputChange}
                name="verifySSLCerts"
                value={verifySSLCerts}
                parseTarget
                tooltip={
                  '<p>Turn this off (not recommended) <br />if you use a self-signed certificate <em className="hint hint--brand"><br />(Default: <strong>On</strong>)</em></p>'
                }
              >
                Verify SSL certs
              </Checkbox>
              <Checkbox
                onChange={handleInputChange}
                name="enableStartTLS"
                value={enableStartTLS}
                parseTarget
                tooltip={
                  '<p>Detects if STARTTLS is enabled <br />in your SMTP server and starts <br />to use it. <em className="hint hint--brand">(Default: <strong>On</strong>)</em></p>'
                }
              >
                Enable STARTTLS
              </Checkbox>
              <Checkbox
                onChange={handleInputChange}
                name="enableHostExpiry"
                value={enableHostExpiry}
                parseTarget
                tooltip={
                  '<p>When enabled, allows automatic cleanup <br />of hosts that have not communicated with Fleet <br />in some number of days. <em className="hint hint--brand">(Default: <strong>Off</strong>)</em></p>'
                }
              >
                Host expiry
              </Checkbox>
              <InputField
                label="Host expiry window"
                type="number"
                disabled={!enableHostExpiry}
                onChange={handleInputChange}
                name="hostExpiryWindow"
                value={hostExpiryWindow}
                parseTarget
                onBlur={validateForm}
                error={formErrors.host_expiry_window}
                tooltip={
                  "<p>If a host has not communicated with Fleet in the specified number of days, it will be removed.</p>"
                }
              />
              <Checkbox
                onChange={handleInputChange}
                name="disableLiveQuery"
                value={disableLiveQuery}
                parseTarget
                tooltip={
                  '<p>When enabled, disables the ability to run live queries <br />(ad hoc queries executed via the UI or fleetctl). <em className="hint hint--brand">(Default: <strong>Off</strong>)</em></p>'
                }
              >
                Disable live queries
              </Checkbox>
            </div>
          </div>
        </div>
      </div>
      <Button
        type="submit"
        variant="brand"
        disabled={Object.keys(formErrors).length > 0}
        className="save-loading"
        spinner={isUpdatingSettings}
      >
        Save
      </Button>
    </form>
  );
};

export default Advanced;
