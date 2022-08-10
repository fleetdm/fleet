import React, { useState, useEffect } from "react";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
// @ts-ignore
import validEmail from "components/forms/validators/valid_email";

import {
  IAppConfigFormProps,
  IFormField,
  IAppConfigFormErrors,
  authMethodOptions,
  authTypeOptions,
} from "../constants";

const baseClass = "app-config-form";

const Smtp = ({
  appConfig,
  handleSubmit,
  isUpdatingSettings,
}: IAppConfigFormProps): JSX.Element => {
  const [formData, setFormData] = useState<any>({
    enableSMTP: appConfig.smtp_settings.enable_smtp || false,
    smtpSenderAddress: appConfig.smtp_settings.sender_address || "",
    smtpServer: appConfig.smtp_settings.server || "",
    smtpPort: appConfig.smtp_settings.port,
    smtpEnableSSLTLS: appConfig.smtp_settings.enable_ssl_tls || false,
    smtpAuthenticationType: appConfig.smtp_settings.authentication_type || "",
    smtpUsername: appConfig.smtp_settings.user_name || "",
    smtpPassword: appConfig.smtp_settings.password || "",
    smtpAuthenticationMethod:
      appConfig.smtp_settings.authentication_method || "",
  });

  const {
    enableSMTP,
    smtpSenderAddress,
    smtpServer,
    smtpPort,
    smtpEnableSSLTLS,
    smtpAuthenticationType,
    smtpUsername,
    smtpPassword,
    smtpAuthenticationMethod,
  } = formData;

  const [formErrors, setFormErrors] = useState<IAppConfigFormErrors>({});

  const handleInputChange = ({ name, value }: IFormField) => {
    setFormData({ ...formData, [name]: value });
  };

  const validateForm = () => {
    const errors: IAppConfigFormErrors = {};

    if (enableSMTP) {
      if (!smtpSenderAddress) {
        errors.sender_address = "SMTP sender address must be present";
      } else if (!validEmail(smtpSenderAddress)) {
        errors.sender_address = `${smtpSenderAddress} is not a valid email`;
      }

      if (!smtpServer) {
        errors.server = "SMTP server must be present";
      }
      if (!smtpPort) {
        errors.server = "SMTP server port must be present";
        errors.server_port = "Port";
      }
      if (!smtpServer && !smtpPort) {
        errors.server = "SMTP server and server port must be present";
        errors.server_port = "Port";
      }
      if (smtpAuthenticationType === "authtype_username_password") {
        if (smtpUsername === "") {
          errors.user_name = "SMTP username must be present";
        }
        if (smtpPassword === "") {
          errors.password = "SMTP password must be present";
        }
      }
    }

    setFormErrors(errors);
  };

  useEffect(() => {
    validateForm();
  }, [smtpAuthenticationType]);

  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    // Formatting of API not UI
    const formDataToSubmit = {
      smtp_settings: {
        enable_smtp: enableSMTP,
        sender_address: smtpSenderAddress,
        server: smtpServer,
        port: Number(smtpPort),
        authentication_type: smtpAuthenticationType,
        user_name: smtpUsername,
        password: smtpPassword,
        enable_ssl_tls: smtpEnableSSLTLS,
        authentication_method: smtpAuthenticationMethod,
        domain: appConfig.smtp_settings.domain || "",
        verify_ssl_certs: appConfig.smtp_settings.verify_ssl_certs || false,
        enable_start_tls: appConfig.smtp_settings.enable_start_tls,
      },
    };

    handleSubmit(formDataToSubmit);
  };

  const renderSmtpSection = () => {
    if (smtpAuthenticationType === "authtype_none") {
      return false;
    }

    return (
      <div className={`${baseClass}__smtp-section`}>
        <InputField
          label="SMTP username"
          onChange={handleInputChange}
          name="smtpUsername"
          value={smtpUsername}
          parseTarget
          onBlur={validateForm}
          error={formErrors.user_name}
          blockAutoComplete
        />
        <InputField
          label="SMTP password"
          type="password"
          onChange={handleInputChange}
          name="smtpPassword"
          value={smtpPassword}
          parseTarget
          onBlur={validateForm}
          error={formErrors.password}
          blockAutoComplete
        />
        <Dropdown
          label="Auth method"
          options={authMethodOptions}
          placeholder=""
          onChange={handleInputChange}
          name="smtpAuthenticationMethod"
          value={smtpAuthenticationMethod}
          parseTarget
        />
      </div>
    );
  };

  return (
    <form className={baseClass} onSubmit={onFormSubmit} autoComplete="off">
      <div className={`${baseClass}__section`}>
        <h2>
          SMTP options{" "}
          <small
            className={`smtp-options smtp-options--${
              appConfig.smtp_settings.configured
                ? "configured"
                : "notconfigured"
            }`}
          >
            STATUS:{" "}
            <em>
              {appConfig.smtp_settings.configured
                ? "CONFIGURED"
                : "NOT CONFIGURED"}
            </em>
          </small>
        </h2>
        <div className={`${baseClass}__inputs`}>
          <Checkbox
            onChange={handleInputChange}
            name="enableSMTP"
            value={enableSMTP}
            parseTarget
          >
            Enable SMTP
          </Checkbox>
        </div>
        <div className={`${baseClass}__inputs`}>
          <InputField
            label="Sender address"
            onChange={handleInputChange}
            name="smtpSenderAddress"
            value={smtpSenderAddress}
            parseTarget
            onBlur={validateForm}
            error={formErrors.sender_address}
            tooltip="The sender address for emails from Fleet."
          />
        </div>
        <div className={`${baseClass}__inputs ${baseClass}__inputs--smtp`}>
          <InputField
            label="SMTP server"
            onChange={handleInputChange}
            name="smtpServer"
            value={smtpServer}
            parseTarget
            onBlur={validateForm}
            error={formErrors.server}
            tooltip="The hostname / IP address and corresponding port of your organization's SMTP server."
          />
          <InputField
            label="&nbsp;"
            type="number"
            onChange={handleInputChange}
            name="smtpPort"
            value={smtpPort}
            parseTarget
            onBlur={validateForm}
            error={formErrors.server_port}
          />
          <Checkbox
            onChange={handleInputChange}
            name="smtpEnableSSLTLS"
            value={smtpEnableSSLTLS}
            parseTarget
          >
            Use SSL/TLS to connect (recommended)
          </Checkbox>
        </div>
        <div className={`${baseClass}__inputs`}>
          <Dropdown
            label="Authentication type"
            options={authTypeOptions}
            onChange={handleInputChange}
            name="smtpAuthenticationType"
            value={smtpAuthenticationType}
            parseTarget
            tooltip={
              "\
              <p>If your mail server requires authentication, you need to specify the authentication type here.</p> \
              <p><strong>No Authentication</strong> - Select this if your SMTP is open.</p> \
              <p><strong>Username & Password</strong> - Select this if your SMTP server requires authentication with a username and password.</p>\
            "
            }
          />
          {renderSmtpSection()}
        </div>
      </div>
      <Button
        type="submit"
        variant="brand"
        disabled={Object.keys(formErrors).length > 0}
        className="save-loading"
        loading={isUpdatingSettings}
      >
        Save
      </Button>
    </form>
  );
};

export default Smtp;
