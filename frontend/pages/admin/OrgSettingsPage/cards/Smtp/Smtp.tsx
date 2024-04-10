import React, { useState, useEffect, useContext } from "react";

import { AppContext } from "context/app";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
// @ts-ignore
import validEmail from "components/forms/validators/valid_email";
import EmptyTable from "components/EmptyTable";
import CustomLink from "components/CustomLink";
import SectionHeader from "components/SectionHeader";

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
  const { isPremiumTier } = useContext(AppContext);

  const [formData, setFormData] = useState<any>({
    enableSMTP: appConfig.smtp_settings?.enable_smtp || false,
    smtpSenderAddress: appConfig.smtp_settings?.sender_address || "",
    smtpServer: appConfig.smtp_settings?.server || "",
    smtpPort: appConfig.smtp_settings?.port,
    smtpEnableSSLTLS: appConfig.smtp_settings?.enable_ssl_tls || false,
    smtpAuthenticationType: appConfig.smtp_settings?.authentication_type || "",
    smtpUsername: appConfig.smtp_settings?.user_name || "",
    smtpPassword: appConfig.smtp_settings?.password || "",
    smtpAuthenticationMethod:
      appConfig.smtp_settings?.authentication_method || "",
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

  const sesConfigured = appConfig.email?.backend === "ses" || false;

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
      },
    };

    handleSubmit(formDataToSubmit);
  };

  const renderSmtpSection = () => {
    if (smtpAuthenticationType === "authtype_none") {
      return false;
    }

    return (
      <>
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
      </>
    );
  };

  const renderSesEnabled = () => {
    const header = "Email already configured";
    const info = (
      <>
        To configure SMTP,{" "}
        <CustomLink
          url={
            isPremiumTier
              ? "https://fleetdm.com/contact"
              : "https://fleetdm.com/slack"
          }
          text="get help"
          newTab
        />
      </>
    );
    return <EmptyTable header={header} info={info} />;
  };

  const renderSmtpForm = () => {
    return (
      <form onSubmit={onFormSubmit} autoComplete="off">
        <Checkbox
          onChange={handleInputChange}
          name="enableSMTP"
          value={enableSMTP}
          parseTarget
        >
          Enable SMTP
        </Checkbox>
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
        <div className="smtp-server-inputs">
          <InputField
            label="SMTP server"
            onChange={handleInputChange}
            name="smtpServer"
            value={smtpServer}
            parseTarget
            onBlur={validateForm}
            error={formErrors.server}
            tooltip="The hostname / private IP address and corresponding port of your organization's SMTP server."
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
        </div>
        <Checkbox
          onChange={handleInputChange}
          name="smtpEnableSSLTLS"
          value={smtpEnableSSLTLS}
          parseTarget
        >
          Use SSL/TLS to connect (recommended)
        </Checkbox>
        <Dropdown
          label="Authentication type"
          options={authTypeOptions}
          onChange={handleInputChange}
          name="smtpAuthenticationType"
          value={smtpAuthenticationType}
          parseTarget
          tooltip={
            <>
              <p>
                If your mail server requires authentication, you need to specify
                the authentication type here.
              </p>
              <p>
                <strong>No Authentication</strong> - Select this if your SMTP is
                open.
              </p>
              <p>
                <strong>Username & Password</strong> - Select this if your SMTP
                server requires authentication with a username and password.
              </p>
            </>
          }
        />
        {renderSmtpSection()}
        <Button
          type="submit"
          variant="brand"
          disabled={Object.keys(formErrors).length > 0}
          className="button-wrap"
          isLoading={isUpdatingSettings}
        >
          Save
        </Button>
      </form>
    );
  };
  return (
    <div className={baseClass}>
      <div className={`${baseClass}__section`}>
        <SectionHeader
          title="SMTP options"
          details={
            !sesConfigured ? (
              <small
                className={`smtp-options smtp-options--${
                  appConfig.smtp_settings?.configured
                    ? "configured"
                    : "notconfigured"
                }`}
              >
                <em>
                  {appConfig.smtp_settings?.configured
                    ? "CONFIGURED"
                    : "NOT CONFIGURED"}
                </em>
              </small>
            ) : (
              <></>
            )
          }
        />
        {sesConfigured ? renderSesEnabled() : renderSmtpForm()}
      </div>
    </div>
  );
};

export default Smtp;
