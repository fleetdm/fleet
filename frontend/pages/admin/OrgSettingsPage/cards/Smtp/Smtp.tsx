import React, { useContext } from "react";

import { AppContext } from "context/app";

import { CONTACT_FLEET_LINK } from "utilities/constants";

import SettingsSection from "pages/admin/components/SettingsSection";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
// @ts-ignore Dropdown is still a JS component
import Dropdown from "components/forms/fields/Dropdown";
import InputField from "components/forms/fields/InputField";
import validEmail from "components/forms/validators/valid_email";
import CustomLink from "components/CustomLink";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import TooltipWrapper from "components/TooltipWrapper";
import Card from "components/Card";

import useFormValidation, { IFormErrors } from "hooks/useFormValidation";

import {
  IAppConfigFormProps,
  authMethodOptions,
  authTypeOptions,
} from "../constants";

interface ISmtpConfigFormData {
  enableSMTP: boolean;
  smtpSenderAddress: string;
  smtpServer: string;
  smtpPort?: number;
  smtpEnableSSLTLS: boolean;
  smtpAuthenticationType: string;
  smtpUsername: string;
  smtpPassword: string;
  smtpAuthenticationMethod: string;
}

const validate = (data: ISmtpConfigFormData): IFormErrors => {
  const errors: IFormErrors = {};

  const {
    enableSMTP,
    smtpSenderAddress,
    smtpServer,
    smtpPort,
    smtpAuthenticationType,
    smtpUsername,
    smtpPassword,
  } = data;

  if (enableSMTP) {
    if (!smtpSenderAddress) {
      errors.smtpSenderAddress = "Enter a sender address";
    } else if (!validEmail(smtpSenderAddress)) {
      errors.smtpSenderAddress = "Enter a valid sender address";
    }

    if (!smtpServer) {
      errors.smtpServer = "Enter an SMTP server";
    }
    if (!smtpPort) {
      errors.smtpPort = "Enter a server port";
    }

    if (smtpAuthenticationType === "authtype_username_password") {
      if (!smtpUsername) {
        errors.smtpUsername = "Enter an SMTP username";
      }
      if (!smtpPassword) {
        errors.smtpPassword = "Enter an SMTP password";
      }
    }
  } else if (smtpSenderAddress && !validEmail(smtpSenderAddress)) {
    // Even when SMTP is disabled, a filled-in sender address must be a valid
    // email so the value is ready when SMTP flips on.
    errors.smtpSenderAddress = "Enter a valid sender address";
  }

  return errors;
};

const baseClass = "app-config-form";

const Smtp = ({
  appConfig,
  handleSubmit,
  isUpdatingSettings,
}: IAppConfigFormProps): JSX.Element => {
  const { isPremiumTier } = useContext(AppContext);
  const gitOpsModeEnabled = appConfig.gitops.gitops_mode_enabled;

  const sesConfigured = appConfig.email?.backend === "ses" || false;

  const {
    formData,
    setField,
    commitFields,
    getError,
    clearFieldError,
    validateField,
    handleSubmit: onSubmit,
    isSubmitting,
  } = useFormValidation<ISmtpConfigFormData>({
    initialFormData: {
      enableSMTP: appConfig.smtp_settings?.enable_smtp || false,
      smtpSenderAddress: appConfig.smtp_settings?.sender_address || "",
      smtpServer: appConfig.smtp_settings?.server || "",
      smtpPort: appConfig.smtp_settings?.port,
      smtpEnableSSLTLS: appConfig.smtp_settings?.enable_ssl_tls || false,
      smtpAuthenticationType:
        appConfig.smtp_settings?.authentication_type || "",
      smtpUsername: appConfig.smtp_settings?.user_name || "",
      smtpPassword: appConfig.smtp_settings?.password || "",
      smtpAuthenticationMethod:
        appConfig.smtp_settings?.authentication_method || "",
    },
    validate,
    isSubmitting: isUpdatingSettings,
    skipTrim: ["smtpPassword"],
  });

  const onValidSubmit = (data: ISmtpConfigFormData) =>
    handleSubmit({
      smtp_settings: {
        enable_smtp: data.enableSMTP,
        sender_address: data.smtpSenderAddress,
        server: data.smtpServer,
        port: Number(data.smtpPort),
        authentication_type: data.smtpAuthenticationType,
        user_name: data.smtpUsername,
        password: data.smtpPassword,
        enable_ssl_tls: data.smtpEnableSSLTLS,
        authentication_method: data.smtpAuthenticationMethod,
      },
    });

  const renderSmtpAuthCredentials = () => {
    if (formData.smtpAuthenticationType === "authtype_none") {
      return null;
    }

    return (
      <>
        <InputField
          label="SMTP username"
          name="smtpUsername"
          value={formData.smtpUsername}
          onChange={(value: string) => setField("smtpUsername", value)}
          onFocus={() => clearFieldError("smtpUsername")}
          onBlur={() => validateField("smtpUsername")}
          error={getError("smtpUsername")}
          blockAutoComplete
          ignore1password={false}
          disabled={isSubmitting}
        />
        <InputField
          label="SMTP password"
          type="password"
          name="smtpPassword"
          value={formData.smtpPassword}
          onChange={(value: string) => setField("smtpPassword", value)}
          onFocus={() => clearFieldError("smtpPassword")}
          onBlur={() => validateField("smtpPassword")}
          error={getError("smtpPassword")}
          blockAutoComplete
          ignore1password={false}
          disabled={isSubmitting}
        />
        <Dropdown
          label="Auth method"
          options={authMethodOptions}
          placeholder=""
          onChange={(value: string) =>
            commitFields({ smtpAuthenticationMethod: value })
          }
          name="smtpAuthenticationMethod"
          value={formData.smtpAuthenticationMethod}
          disabled={isSubmitting}
        />
      </>
    );
  };

  const renderSesEnabled = () => {
    const sesBaseClass = `${baseClass}__ses-enabled`;
    return (
      <Card paddingSize="xxlarge" className={sesBaseClass}>
        <div className={`${sesBaseClass}__content`}>
          <p className={`${sesBaseClass}__title`}>Email already configured</p>
          <p>
            To configure SMTP,{" "}
            <CustomLink
              url={
                isPremiumTier ? CONTACT_FLEET_LINK : "https://fleetdm.com/slack"
              }
              text="get help"
              newTab
            />
          </p>
        </div>
      </Card>
    );
  };

  const renderSmtpForm = () => {
    return (
      <form onSubmit={onSubmit(onValidSubmit)} autoComplete="off">
        <div
          className={`form ${
            gitOpsModeEnabled ? "disabled-by-gitops-mode" : ""
          }`}
        >
          <Checkbox
            name="enableSMTP"
            value={formData.enableSMTP}
            onChange={(value: boolean) => commitFields({ enableSMTP: value })}
            disabled={isSubmitting}
          >
            Enable SMTP
          </Checkbox>
          <InputField
            label="Sender address"
            name="smtpSenderAddress"
            value={formData.smtpSenderAddress}
            onChange={(value: string) => setField("smtpSenderAddress", value)}
            onFocus={() => clearFieldError("smtpSenderAddress")}
            onBlur={() => validateField("smtpSenderAddress")}
            error={getError("smtpSenderAddress")}
            tooltip="The sender address for emails from Fleet."
            disabled={isSubmitting}
          />
          <div className="smtp-server-inputs">
            <InputField
              label="SMTP server"
              name="smtpServer"
              value={formData.smtpServer}
              onChange={(value: string) => setField("smtpServer", value)}
              onFocus={() => clearFieldError("smtpServer")}
              onBlur={() => validateField("smtpServer")}
              error={getError("smtpServer")}
              tooltip="The hostname / private IP address and corresponding port of your organization's SMTP server."
              disabled={isSubmitting}
            />
            <InputField
              label="&nbsp;"
              type="number"
              name="smtpPort"
              value={formData.smtpPort}
              onChange={(value: string) =>
                setField("smtpPort", value ? Number(value) : undefined)
              }
              onFocus={() => clearFieldError("smtpPort")}
              onBlur={() => validateField("smtpPort")}
              error={getError("smtpPort")}
              disabled={isSubmitting}
            />
          </div>
          <Checkbox
            name="smtpEnableSSLTLS"
            value={formData.smtpEnableSSLTLS}
            onChange={(value: boolean) =>
              commitFields({ smtpEnableSSLTLS: value })
            }
            labelTooltipContent={
              <>
                To disable this setting, STARTTLS must first be disabled in{" "}
                <strong>Organization settings</strong> &gt;{" "}
                <strong>Advanced options</strong>.
              </>
            }
            disabled={isSubmitting}
          >
            Use SSL/TLS to connect (recommended)
          </Checkbox>
          <Dropdown
            label="Authentication type"
            options={authTypeOptions}
            onChange={(value: string) =>
              commitFields({ smtpAuthenticationType: value })
            }
            name="smtpAuthenticationType"
            value={formData.smtpAuthenticationType}
            disabled={isSubmitting}
            tooltip={
              <>
                If your mail server requires authentication, you need to specify
                the authentication type here.
                <br />
                <br />
                <strong>No Authentication</strong> - Select this if your SMTP is
                open.
                <br />
                <br />
                <strong>Username & Password</strong> - Select this if your SMTP
                server requires authentication with a username and password.
              </>
            }
          />
          {renderSmtpAuthCredentials()}
        </div>
        <GitOpsModeTooltipWrapper
          renderChildren={(disableChildren) => (
            <TooltipWrapper
              tipContent={
                disableChildren ? "" : "Saving changes will send a test email"
              }
              position="right"
              className="button-wrap"
              tipOffset={8}
              showArrow
              underline={false}
            >
              <Button
                type="submit"
                disabled={disableChildren || isSubmitting}
                isLoading={isSubmitting}
              >
                Save
              </Button>
            </TooltipWrapper>
          )}
        />
      </form>
    );
  };
  return (
    <SettingsSection title="SMTP options">
      {sesConfigured ? renderSesEnabled() : renderSmtpForm()}
    </SettingsSection>
  );
};

export default Smtp;
