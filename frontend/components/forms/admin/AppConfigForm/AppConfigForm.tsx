import React, { useState, useEffect } from "react";
import { syntaxHighlight } from "fleet/helpers";

// @ts-ignore
import constructErrorString from "utilities/yaml";

import { IConfigFormData } from "interfaces/config";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import EnrollSecretTable from "components/EnrollSecretTable";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
// @ts-ignore
import OrgLogoIcon from "components/icons/OrgLogoIcon";
// @ts-ignore
import validateYaml from "components/forms/validators/validate_yaml";
import IconToolTip from "components/IconToolTip";
import InfoBanner from "components/InfoBanner/InfoBanner";
// @ts-ignore
import YamlAce from "components/YamlAce";
import Modal from "components/Modal";
import SelectTargetsDropdownStories from "components/forms/fields/SelectTargetsDropdown/SelectTargetsDropdown.stories";
import OpenNewTabIcon from "../../../../../assets/images/open-new-tab-12x12@2x.png";
import {
  IAppConfigFormProps,
  IFormField,
  IAppConfigFormErrors,
  authMethodOptions,
  authTypeOptions,
  percentageOfHosts,
  numberOfDays,
  hostStatusPreview,
  usageStatsPreview,
} from "./constants";

const baseClass = "app-config-form";

const AppConfigFormFunctional = ({
  appConfig,
  enrollSecret,
  handleSubmit,
}: IAppConfigFormProps): JSX.Element => {
  // STATE
  const [
    showHostStatusWebhookPreviewModal,
    setShowHostStatusWebhookPreviewModal,
  ] = useState<boolean>(false);
  const [
    showUsageStatsPreviewModal,
    setShowUsageStatsPreviewModal,
  ] = useState<boolean>(false);

  // FORM STATE
  const [formData, setFormData] = useState<IConfigFormData>({
    // Formatting of UI not API
    // Organization info
    orgName: appConfig.org_info.org_name || "",
    orgLogoURL: appConfig.org_info.org_logo_url || "",
    // Fleet web address
    serverURL: appConfig.server_settings.server_url || "",
    // SAML single sign on options
    enableSSO: appConfig.sso_settings.enable_sso || false,
    idpName: appConfig.sso_settings.idp_name || "",
    entityID: appConfig.sso_settings.entity_id || "",
    issuerURI: appConfig.sso_settings.issuer_uri || "",
    idpImageURL: appConfig.sso_settings.idp_image_url || "",
    metadata: appConfig.sso_settings.metadata || "",
    metadataURL: appConfig.sso_settings.metadata_url || "",
    enableSSOIDPLogin: appConfig.sso_settings.enable_sso_idp_login || false,
    // SMTP options
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
    // Global agent options
    agentOptions: appConfig.agent_options || {},
    // Host status webhook
    enableHostStatusWebhook:
      appConfig.webhook_settings.host_status_webhook
        .enable_host_status_webhook || false,
    hostStatusWebhookDestinationURL:
      appConfig.webhook_settings.host_status_webhook.destination_url || "",
    hostStatusWebhookHostPercentage:
      appConfig.webhook_settings.host_status_webhook.host_percentage ||
      undefined,
    hostStatusWebhookDaysCount:
      appConfig.webhook_settings.host_status_webhook.days_count || undefined,
    // Usage statistics
    enableUsageStatistics: appConfig.server_settings.enable_analytics,
    // Advanced options
    domain: appConfig.smtp_settings.domain || "",
    verifySSLCerts: appConfig.smtp_settings.verify_ssl_certs || false,
    enableStartTLS: appConfig.smtp_settings.enable_start_tls,
    enableHostExpiry:
      appConfig.host_expiry_settings.host_expiry_enabled || false,
    hostExpiryWindow: appConfig.host_expiry_settings.host_expiry_window || 0,
    disableLiveQuery: appConfig.server_settings.live_query_disabled || false,
  });

  const {
    orgName,
    orgLogoURL,
    serverURL,
    enableSSO,
    idpName,
    entityID,
    issuerURI,
    idpImageURL,
    metadata,
    metadataURL,
    enableSSOIDPLogin,
    enableSMTP,
    smtpSenderAddress,
    smtpServer,
    smtpPort,
    smtpEnableSSLTLS,
    smtpAuthenticationType,
    smtpUsername,
    smtpPassword,
    smtpAuthenticationMethod,
    agentOptions,
    enableHostStatusWebhook,
    hostStatusWebhookDestinationURL,
    hostStatusWebhookHostPercentage,
    hostStatusWebhookDaysCount,
    enableUsageStatistics,
    domain,
    verifySSLCerts,
    enableStartTLS,
    enableHostExpiry,
    hostExpiryWindow,
    disableLiveQuery,
  } = formData;

  const [formErrors, setFormErrors] = useState<IAppConfigFormErrors>({});

  // FORM CHANGE AND VALIDATIONS
  const handleInputChange = ({ name, value }: IFormField) => {
    setFormData({ ...formData, [name]: value });
  };

  const handleAceInputChange = (value: string) => {
    setFormData({ ...formData, agentOptions: value });
  };

  const validateForm = () => {
    const errors: IAppConfigFormErrors = {};

    if (!orgName) {
      errors.org_name = "Organization name must be present";
    }

    if (!serverURL) {
      errors.server_url = "Fleet server URL must be present";
    }

    if (enableSSO) {
      if (metadata === "" && metadataURL === "") {
        errors.metadata_url = "Metadata URL must be present";
      }
      if (!entityID) {
        errors.entity_id = "Entity ID must be present";
      }
      if (!idpName) {
        errors.idp_name = "Identity provider name must be present";
      }
    }

    if (enableSMTP) {
      if (!smtpSenderAddress) {
        errors.sender_address = "SMTP sender address must be present";
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

    if (enableHostStatusWebhook) {
      if (!hostStatusWebhookDestinationURL) {
        errors.destination_url = "Destination URL must be present";
      }
    }

    if (enableHostExpiry) {
      if (!hostExpiryWindow) {
        errors.host_expiry_window =
          "Host expiry window must be a positive number";
      }
    }

    if (agentOptions) {
      const { error: yamlError, valid: yamlValid } = validateYaml(agentOptions);
      if (!yamlValid) {
        errors.agent_options = constructErrorString(yamlError);
      }
    }

    setFormErrors(errors);
  };

  // Validates forms when certain checkboxes and dropdowns are selected
  useEffect(() => {
    validateForm();
  }, [
    enableSSO,
    enableSMTP,
    smtpAuthenticationType,
    enableHostStatusWebhook,
    enableHostExpiry,
    agentOptions,
  ]);

  // TOGGLE MODALS

  const toggleHostStatusWebhookPreviewModal = () => {
    setShowHostStatusWebhookPreviewModal(!showHostStatusWebhookPreviewModal);
    return false;
  };

  const toggleUsageStatsPreviewModal = () => {
    setShowUsageStatsPreviewModal(!showUsageStatsPreviewModal);
    return false;
  };

  // FORM SUBMIT
  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    // Formatting of API not UI
    const formDataToSubmit = {
      org_info: {
        org_logo_url: orgLogoURL,
        org_name: orgName,
      },
      server_settings: {
        server_url: serverURL,
        live_query_disabled: disableLiveQuery,
        enable_analytics: enableUsageStatistics,
      },
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
        domain,
        verify_ssl_certs: verifySSLCerts,
        enable_start_tls: enableStartTLS,
      },
      sso_settings: {
        entity_id: entityID,
        issuer_uri: issuerURI,
        idp_image_url: idpImageURL,
        metadata,
        metadata_url: metadataURL,
        idp_name: idpName,
        enable_sso: enableSSO,
        enable_sso_idp_login: enableSSOIDPLogin,
      },
      host_expiry_settings: {
        host_expiry_enabled: enableHostExpiry,
        host_expiry_window: Number(hostExpiryWindow),
      },
      agent_options: agentOptions,
      webhook_settings: {
        host_status_webhook: {
          enable_host_status_webhook: enableHostStatusWebhook,
          destination_url: hostStatusWebhookDestinationURL,
          host_percentage: hostStatusWebhookHostPercentage,
          days_count: hostStatusWebhookDaysCount,
        },
      },
    };

    handleSubmit(formDataToSubmit);
  };

  // SECTIONS
  const renderOrganizationInfoSection = () => {
    return (
      <div className={`${baseClass}__section`}>
        <h2>
          <a id="organization-info">Organization info</a>
        </h2>
        <div className={`${baseClass}__inputs`}>
          <InputField
            label="Organization name"
            onChange={handleInputChange}
            name="orgName"
            value={orgName}
            parseTarget
            onBlur={validateForm}
            error={formErrors.org_name}
          />
          <InputField
            label="Organization avatar URL"
            onChange={handleInputChange}
            name="orgLogoURL"
            value={orgLogoURL}
            parseTarget
          />
        </div>
        <div className={`${baseClass}__details ${baseClass}__avatar-preview`}>
          <OrgLogoIcon src={orgLogoURL} />
        </div>
      </div>
    );
  };

  const renderFleetWebAddressSection = () => {
    return (
      <div className={`${baseClass}__section`}>
        <h2>
          <a id="fleet-web-address">Fleet web address</a>
        </h2>
        <div className={`${baseClass}__inputs`}>
          <InputField
            label="Fleet app URL"
            hint={
              <span>
                Include base path only (eg. no <code>/v1</code>)
              </span>
            }
            onChange={handleInputChange}
            name="serverURL"
            value={serverURL}
            parseTarget
            onBlur={validateForm}
            error={formErrors.server_url}
          />
        </div>
        <div className={`${baseClass}__details`}>
          <IconToolTip
            text={"The base URL of this instance for use in Fleet links."}
          />
        </div>
      </div>
    );
  };

  const renderSAMLSingleSignOnOptionsSection = () => {
    return (
      <div className={`${baseClass}__section`}>
        <h2>
          <a id="saml">SAML single sign on options</a>
        </h2>
        <div className={`${baseClass}__inputs`}>
          <Checkbox
            onChange={handleInputChange}
            name="enableSSO"
            value={enableSSO}
            parseTarget
          >
            Enable single sign on
          </Checkbox>
        </div>
        <div className={`${baseClass}__inputs`}>
          <InputField
            label="Identity provider name"
            onChange={handleInputChange}
            name="idpName"
            value={idpName}
            parseTarget
            onBlur={validateForm}
            error={formErrors.idp_name}
          />
        </div>
        <div className={`${baseClass}__details`}>
          <IconToolTip
            text={
              "A required human friendly name for the identity provider that will provide single sign on authentication."
            }
          />
        </div>
        <div className={`${baseClass}__inputs`}>
          <InputField
            label="Entity ID"
            hint={
              <span>
                The URI you provide here must exactly match the Entity ID field
                used in identity provider configuration.
              </span>
            }
            onChange={handleInputChange}
            name="entityID"
            value={entityID}
            parseTarget
            onBlur={validateForm}
            error={formErrors.entity_id}
          />
        </div>
        <div className={`${baseClass}__details`}>
          <IconToolTip
            text={
              "The required entity ID is a URI that you use to identify Fleet when configuring the identity provider."
            }
          />
        </div>
        <div className={`${baseClass}__inputs`}>
          <InputField
            label="Issuer URI"
            onChange={handleInputChange}
            name="issuerURI"
            value={issuerURI}
            parseTarget
          />
        </div>
        <div className={`${baseClass}__details`}>
          <IconToolTip
            text={"The issuer URI supplied by the identity provider."}
          />
        </div>
        <div className={`${baseClass}__inputs`}>
          <InputField
            label="IDP image URL"
            onChange={handleInputChange}
            name="idpImageURL"
            value={idpImageURL}
            parseTarget
          />
        </div>
        <div className={`${baseClass}__details`}>
          <IconToolTip
            text={
              "An optional link to an image such as a logo for the identity provider."
            }
          />
        </div>
        <div className={`${baseClass}__inputs`}>
          <InputField
            label="Metadata"
            type="textarea"
            onChange={handleInputChange}
            name="metadata"
            value={metadata}
            parseTarget
            onBlur={validateForm}
          />
        </div>
        <div className={`${baseClass}__details`}>
          <IconToolTip
            text={
              "Metadata provided by the identity provider. Either metadata or a metadata url must be provided."
            }
          />
        </div>
        <div className={`${baseClass}__inputs`}>
          <InputField
            label="Metadata URL"
            hint={
              <span>
                If available from the identity provider, this is the preferred
                means of providing metadata.
              </span>
            }
            onChange={handleInputChange}
            name="metadataURL"
            value={metadataURL}
            parseTarget
            onBlur={validateForm}
            error={formErrors.metadata_url}
          />
        </div>
        <div className={`${baseClass}__details`}>
          <IconToolTip
            text={"A URL that references the identity provider metadata."}
          />
        </div>

        <div className={`${baseClass}__inputs`}>
          <Checkbox
            onChange={handleInputChange}
            name="enableSSOIDPLogin"
            value={enableSSOIDPLogin}
            parseTarget
          >
            Allow SSO login initiated by Identity Provider
          </Checkbox>
        </div>
      </div>
    );
  };

  const renderSMTPOptionsSection = () => {
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
      <div className={`${baseClass}__section`}>
        <h2>
          <a id="smtp">
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
          </a>
        </h2>
        <div className={`${baseClass}__inputs`}>
          <Checkbox
            onChange={handleInputChange}
            onFocus={validateForm}
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
          />
        </div>
        <div className={`${baseClass}__details`}>
          <IconToolTip text={"The sender address for emails from Fleet."} />
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
        <div className={`${baseClass}__details`}>
          <IconToolTip
            text={
              "The hostname / IP address and corresponding port of your organization's SMTP server."
            }
          />
        </div>
        <div className={`${baseClass}__inputs`}>
          <Dropdown
            label="Authentication type"
            options={authTypeOptions}
            onChange={handleInputChange}
            name="smtpAuthenticationType"
            value={smtpAuthenticationType}
            parseTarget
          />
          {renderSmtpSection()}
        </div>
        <div className={`${baseClass}__details`}>
          <IconToolTip
            isHtml
            text={
              "\
                  <p>If your mail server requires authentication, you need to specify the authentication type here.</p> \
                  <p><strong>No Authentication</strong> - Select this if your SMTP is open.</p> \
                  <p><strong>Username & Password</strong> - Select this if your SMTP server requires authentication with a username and password.</p>\
                "
            }
          />
        </div>
      </div>
    );
  };

  const renderOsqueryEnrollmentSecretsSection = () => {
    return (
      <div className={`${baseClass}__section`}>
        <h2>
          <a id="osquery-enrollment-secrets">Osquery enrollment secrets</a>
        </h2>
        <div className={`${baseClass}__inputs`}>
          <p className={`${baseClass}__enroll-secret-label`}>
            Manage secrets with <code>fleetctl</code>. Active secrets:
          </p>
          <EnrollSecretTable secrets={enrollSecret} />
        </div>
      </div>
    );
  };

  const renderGlobalAgentOptionsSection = () => {
    return (
      <div className={`${baseClass}__section`}>
        <h2>
          <a id="agent-options">Global agent options</a>
        </h2>
        <div className={`${baseClass}__yaml`}>
          <p className={`${baseClass}__section-description`}>
            This code will be used by osquery when it checks for configuration
            options.
            <br />
            <b>
              Changes to these configuration options will be applied to all
              hosts in your organization that do not belong to any team.
            </b>
          </p>
          <InfoBanner className={`${baseClass}__config-docs`}>
            How do global agent options interact with team-level agent
            options?&nbsp;
            <a
              href="https://github.com/fleetdm/fleet/blob/2f42c281f98e39a72ab4a5125ecd26d303a16a6b/docs/1-Using-Fleet/1-Fleet-UI.md#configuring-agent-options"
              className={`${baseClass}__learn-more ${baseClass}__learn-more--inline`}
              target="_blank"
              rel="noopener noreferrer"
            >
              Learn more about agent options&nbsp;
              <img className="icon" src={OpenNewTabIcon} alt="open new tab" />
            </a>
          </InfoBanner>
          <p className={`${baseClass}__component-label`}>
            <b>YAML</b>
          </p>
          {/* <GlobalAgentOptions fields={agentOptions} /> */}
          <YamlAce
            wrapperClassName={`${baseClass}__text-editor-wrapper`}
            onChange={handleAceInputChange}
            name="agentOptions" // TODO
            value={agentOptions} // TODO
            parseTarget
            onBlur={validateForm}
            error={formErrors.agent_options}
          />
        </div>
      </div>
    );
  };

  const renderHostStatusWebhookSection = () => {
    return (
      <div className={`${baseClass}__section`}>
        <h2>
          <a id="host-status-webhook">Host status webhook</a>
        </h2>
        <div className={`${baseClass}__host-status-webhook`}>
          <p className={`${baseClass}__section-description`}>
            Send an alert if a portion of your hosts go offline.
          </p>
          <Checkbox
            onChange={handleInputChange}
            name="enableHostStatusWebhook"
            value={enableHostStatusWebhook}
            parseTarget
          >
            Enable host status webhook
          </Checkbox>
          <p className={`${baseClass}__section-description`}>
            A request will be sent to your configured <b>Destination URL</b> if
            the configured <b>Percentage of hosts</b> have not checked into
            Fleet for the configured <b>Number of days</b>.
          </p>
        </div>
        <div className={`${baseClass}__inputs ${baseClass}__inputs--webhook`}>
          <Button
            type="button"
            variant="inverse"
            onClick={toggleHostStatusWebhookPreviewModal}
          >
            Preview request
          </Button>
        </div>
        <div className={`${baseClass}__inputs`}>
          <InputField
            placeholder="https://server.com/example"
            label="Destination URL"
            onChange={handleInputChange}
            name="hostStatusWebhookDestinationURL"
            value={hostStatusWebhookDestinationURL}
            parseTarget
            onBlur={validateForm}
            error={formErrors.destination_url}
          />
        </div>
        <div className={`${baseClass}__details`}>
          <IconToolTip
            isHtml
            text={
              "\
                  <center><p>Provide a URL to deliver <br/>the webhook request to.</p></center>\
                "
            }
          />
        </div>
        <div className={`${baseClass}__inputs ${baseClass}__host-percentage`}>
          <Dropdown
            label="Percentage of hosts"
            options={percentageOfHosts}
            onChange={handleInputChange}
            name="hostStatusWebhookHostPercentage"
            value={hostStatusWebhookHostPercentage}
            parseTarget
          />
        </div>
        <div className={`${baseClass}__details`}>
          <IconToolTip
            isHtml
            text={
              "\
                  <center><p>Select the minimum percentage of hosts that<br/>must fail to check into Fleet in order to trigger<br/>the webhook request.</p></center>\
                "
            }
          />
        </div>
        <div className={`${baseClass}__inputs ${baseClass}__days-count`}>
          <Dropdown
            label="Number of days"
            options={numberOfDays}
            onChange={handleInputChange}
            name="hostStatusWebhookDaysCount"
            value={hostStatusWebhookDaysCount}
            parseTarget
          />
        </div>
        <div className={`${baseClass}__details`}>
          <IconToolTip
            isHtml
            text={
              "\
                  <center><p>Select the minimum number of days that the<br/>configured <b>Percentage of hosts</b> must fail to<br/>check into Fleet in order to trigger the<br/>webhook request.</p></center>\
                "
            }
          />
        </div>
      </div>
    );
  };

  const renderUsageStatistics = () => {
    return (
      <div className={`${baseClass}__section`}>
        <h2>
          <a id="usage-stats">Usage statistics</a>
        </h2>
        <p className={`${baseClass}__section-description`}>
          Help improve Fleet by sending anonymous usage statistics.
          <br />
          <br />
          This information helps our team better understand feature adoption and
          usage, and allows us to see how Fleet is adding value, so that we can
          make better product decisions.
          <br />
          <br />
          <a
            href="https://github.com/fleetdm/fleet/blob/2f42c281f98e39a72ab4a5125ecd26d303a16a6b/docs/1-Using-Fleet/11-Usage-statistics.md"
            className={`${baseClass}__learn-more`}
            target="_blank"
            rel="noopener noreferrer"
          >
            Learn more about usage statistics&nbsp;
            <img className="icon" src={OpenNewTabIcon} alt="open new tab" />
          </a>
        </p>
        <div className={`${baseClass}__inputs ${baseClass}__inputs--usage`}>
          <Checkbox
            onChange={handleInputChange}
            name="enableUsageStatistics"
            value={enableUsageStatistics}
            parseTarget
          >
            Enable usage statistics
          </Checkbox>
        </div>
        <div className={`${baseClass}__inputs ${baseClass}__inputs--usage`}>
          <Button
            type="button"
            variant="inverse"
            onClick={toggleUsageStatsPreviewModal}
          >
            Preview payload
          </Button>
        </div>
      </div>
    );
  };

  const renderAdvancedOptions = () => {
    return (
      <div className={`${baseClass}__section`}>
        <h2>
          <a id="advanced-options">Advanced options</a>
        </h2>
        <div className={`${baseClass}__advanced-options`}>
          <p className={`${baseClass}__section-description`}>
            Most users do not need to modify these options.
          </p>
          <div className={`${baseClass}__inputs`}>
            <div className={`${baseClass}__form-fields`}>
              <div className="tooltip-wrap tooltip-wrap--input">
                <InputField
                  label="Domain"
                  onChange={handleInputChange}
                  name="domain"
                  value={domain}
                  parseTarget
                />
                <IconToolTip
                  isHtml
                  text={
                    '<p>If you need to specify a HELO domain, <br />you can do it here <em className="hint hint--brand">(Default: <strong>Blank</strong>)</em></p>'
                  }
                />
              </div>
              <div className="tooltip-wrap">
                <Checkbox
                  onChange={handleInputChange}
                  name="verifySSLCerts"
                  value={verifySSLCerts}
                  parseTarget
                >
                  Verify SSL certs
                </Checkbox>
                <IconToolTip
                  isHtml
                  text={
                    '<p>Turn this off (not recommended) <br />if you use a self-signed certificate <em className="hint hint--brand"><br />(Default: <strong>On</strong>)</em></p>'
                  }
                />
              </div>
              <div className="tooltip-wrap">
                <Checkbox
                  onChange={handleInputChange}
                  name="enableStartTLS"
                  value={enableStartTLS}
                  parseTarget
                >
                  Enable STARTTLS
                </Checkbox>
                <IconToolTip
                  isHtml
                  text={
                    '<p>Detects if STARTTLS is enabled <br />in your SMTP server and starts <br />to use it. <em className="hint hint--brand">(Default: <strong>On</strong>)</em></p>'
                  }
                />
              </div>
              <div className="tooltip-wrap">
                <Checkbox
                  onChange={handleInputChange}
                  name="enableHostExpiry"
                  value={enableHostExpiry}
                  parseTarget
                >
                  Host expiry
                </Checkbox>
                <IconToolTip
                  isHtml
                  text={
                    '<p>When enabled, allows automatic cleanup <br />of hosts that have not communicated with Fleet <br />in some number of days. <em className="hint hint--brand">(Default: <strong>Off</strong>)</em></p>'
                  }
                />
              </div>
              <div className="tooltip-wrap tooltip-wrap--input">
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
                />
                <IconToolTip
                  isHtml
                  text={
                    "<p>If a host has not communicated with Fleet <br />in the specified number of days, it will be removed.</p>"
                  }
                />
              </div>
              <div className="tooltip-wrap">
                <Checkbox
                  onChange={handleInputChange}
                  name="disableLiveQuery"
                  value={disableLiveQuery}
                  parseTarget
                >
                  Disable live queries
                </Checkbox>
                <IconToolTip
                  isHtml
                  text={
                    '<p>When enabled, disables the ability to run live queries <br />(ad hoc queries executed via the UI or fleetctl). <em className="hint hint--brand">(Default: <strong>Off</strong>)</em></p>'
                  }
                />
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  };

  // MODALS

  const renderHostStatusWebhookPreviewModal = () => {
    if (!showHostStatusWebhookPreviewModal) {
      return null;
    }

    return (
      <Modal
        title="Host status webhook"
        onExit={toggleHostStatusWebhookPreviewModal}
        className={`${baseClass}__host-status-webhook-preview-modal`}
      >
        <>
          <p>
            An example request sent to your configured <b>Destination URL</b>.
          </p>
          <div className={`${baseClass}__host-status-webhook-preview`}>
            <pre
              dangerouslySetInnerHTML={{
                __html: syntaxHighlight(hostStatusPreview),
              }}
            />
          </div>
          <div className="flex-end">
            <Button type="button" onClick={toggleHostStatusWebhookPreviewModal}>
              Done
            </Button>
          </div>
        </>
      </Modal>
    );
  };

  const renderUsageStatsPreviewModal = () => {
    if (!showUsageStatsPreviewModal) {
      return null;
    }

    return (
      <Modal
        title="Usage statistics"
        onExit={toggleUsageStatsPreviewModal}
        className={`${baseClass}__usage-stats-preview-modal`}
      >
        <>
          <p>An example JSON payload sent to Fleet Device Management Inc.</p>
          <pre
            dangerouslySetInnerHTML={{
              __html: syntaxHighlight(usageStatsPreview),
            }}
          />
          <div className="flex-end">
            <Button type="button" onClick={toggleUsageStatsPreviewModal}>
              Done
            </Button>
          </div>
        </>
      </Modal>
    );
  };

  // RENDER
  return (
    <>
      <form className={baseClass} onSubmit={onFormSubmit} autoComplete="off">
        {renderOrganizationInfoSection()}
        {renderFleetWebAddressSection()}
        {renderSAMLSingleSignOnOptionsSection()}
        {renderSMTPOptionsSection()}
        {renderOsqueryEnrollmentSecretsSection()}
        {renderGlobalAgentOptionsSection()}
        {renderHostStatusWebhookSection()}
        {renderUsageStatistics()}
        {renderAdvancedOptions()}
        <Button
          type="submit"
          variant="brand"
          disabled={Object.keys(formErrors).length > 0}
        >
          Update settings
        </Button>
      </form>
      {renderUsageStatsPreviewModal()}
      {renderHostStatusWebhookPreviewModal()}
    </>
  );
};

export default AppConfigFormFunctional;
