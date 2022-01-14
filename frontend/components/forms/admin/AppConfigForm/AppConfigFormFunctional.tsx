import React, { useState, useCallback } from "react";
import { syntaxHighlight } from "fleet/helpers";
import { size } from "lodash";

import yaml from "js-yaml";

// @ts-ignore
import constructErrorString from "utilities/yaml";

import { IConfigNested, IConfigFormData } from "interfaces/config";
import { IEnrollSecret } from "interfaces/enroll_secret";

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
// @ts-ignore
import validate from "components/forms/admin/AppConfigForm/validate";
import IconToolTip from "components/IconToolTip";
import InfoBanner from "components/InfoBanner/InfoBanner";
// @ts-ignore
import YamlAce from "components/YamlAce";
import Modal from "components/Modal";
import { string } from "prop-types";
import SelectTargetsDropdownStories from "components/forms/fields/SelectTargetsDropdown/SelectTargetsDropdown.stories";
import OpenNewTabIcon from "../../../../../assets/images/open-new-tab-12x12@2x.png";

const authMethodOptions = [
  { label: "Plain", value: "authmethod_plain" },
  { label: "Cram MD5", value: "authmethod_cram_md5" },
  { label: "Login", value: "authmethod_login" },
];
const authTypeOptions = [
  { label: "Username and Password", value: "authtype_username_password" },
  { label: "None", value: "authtype_none" },
];
const percentageOfHosts = [
  { label: "1%", value: 1 },
  { label: "5%", value: 5 },
  { label: "10%", value: 10 },
  { label: "25%", value: 25 },
];
const numberOfDays = [
  { label: "1 day", value: 1 },
  { label: "3 days", value: 3 },
  { label: "7 days", value: 7 },
  { label: "14 days", value: 14 },
];

// TODO: consider breaking this up into separate components/files

const baseClass = "app-config-form";

interface IAppConfigFormProps {
  formData: IConfigNested;
  enrollSecret: IEnrollSecret[] | undefined;
  handleSubmit: any;
}

interface IFormField {
  name: string;
  value: string | boolean | number;
}

interface IAppConfigFormErrors {
  metadata_url?: string | null;
  entity_id?: string | null;
  idp_name?: string | null;
  server_url?: string | null;
  org_name?: string | null;
  sender_address?: string | null;
  server?: string | null;
  user_name?: string | null;
  password?: string | null;
  destination_url?: string | null;
  host_percentage?: string | null;
  days_count?: string | null;
  host_expiry_window?: string | null;
  agent_options?: string | null;
}

const AppConfigFormFunctional = ({
  formData,
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
  const [iterateFormData, setIterateFormData] = useState<any>({
    // Organization info
    orgName: formData.org_info.org_name || "",
    orgLogoURL: formData.org_info.org_logo_url || "",
    // Fleet web address
    serverURL: formData.server_settings.server_url || "",
    // SAML single sign on options
    enableSSO: formData.sso_settings.enable_sso || false,
    idpName: formData.sso_settings.idp_name || "",
    entityID: formData.sso_settings.entity_id || "",
    issuerURI: formData.sso_settings.issuer_uri || "",
    idpImageURL: formData.sso_settings.idp_image_url || "",
    metadata: formData.sso_settings.metadata || "",
    metadataURL: formData.sso_settings.metadata_url || "",
    enableSSOIDPLogin: formData.sso_settings.enable_sso_idp_login || false,
    // SMTP options
    enableSMTP: formData.smtp_settings.enable_smtp || false,
    smtpSenderAddress: formData.smtp_settings.sender_address || "",
    smtpServer: formData.smtp_settings.server || "",
    smtpPort: formData.smtp_settings.port,
    smtpEnableSSLTLS: formData.smtp_settings.enable_ssl_tls || false,
    smtpAuthenticationType: formData.smtp_settings.authentication_type || "",
    smtpUsername: formData.smtp_settings.user_name || "",
    smtpPassword: formData.smtp_settings.password || "",
    smtpAuthenticationMethod:
      formData.smtp_settings.authentication_method || "",
    // Global agent options
    agentOptions: yaml.dump(formData.agent_options) || {},
    // Host status webhook
    enableHostStatusWebhook:
      formData.webhook_settings.host_status_webhook
        .enable_host_status_webhook || false,
    hostStatusWebhookDestinationURL:
      formData.webhook_settings.host_status_webhook.destination_url || "",
    hostStatusWebhookHostPercentage:
      formData.webhook_settings.host_status_webhook.host_percentage ||
      undefined,
    hostStatusWebhookDaysCount:
      formData.webhook_settings.host_status_webhook.days_count || undefined,
    // Usage statistics
    enableUsageStatistics: formData.server_settings.enable_analytics,
    // Advanced options
    domain: formData.smtp_settings.domain || "",
    verifySSLCerts: formData.smtp_settings.verify_ssl_certs || false,
    enableStartTLS: formData.smtp_settings.enable_start_tls,
    enableHostExpiry:
      formData.host_expiry_settings.host_expiry_enabled || false,
    hostExpiryWindow: formData.host_expiry_settings.host_expiry_window || 0,
    disableLiveQuery: formData.server_settings.live_query_disabled || false,
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
  } = iterateFormData;

  // OLD
  const [formErrors, setFormErrors] = useState<IAppConfigFormErrors>({});

  // FORM CHANGE
  const handleInputChange = ({ name, value }: IFormField) => {
    console.log("name", name);
    console.log("value", value);
    setIterateFormData({ ...iterateFormData, [name]: value });
  };

  console.log("iterateFormData", iterateFormData);

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
  const onFormSubmit = () => {
    // Validators
    const errors: any = {};

    if (enableSSO) {
      if (!metadata && !metadataURL) {
        errors.metadata_url = "Metadata URL must be present";
      }
      if (!entityID) {
        errors.entity_id = "Entity ID must be present";
      }
      if (!idpName) {
        errors.idp_name = "Identity Provider Name must be present";
      }
    }

    if (!serverURL) {
      errors.server_url = "Fleet server URL must be present";
    }

    if (!orgName) {
      errors.org_name = "Organization name must be present";
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
      }

      if (smtpAuthenticationType !== "authtype_none") {
        if (!smtpUsername) {
          errors.user_name = "SMTP username must be present";
        }

        if (!smtpPassword) {
          errors.password = "SMTP password must be present";
        }
      }
    }

    if (enableHostStatusWebhook) {
      if (!hostStatusWebhookDestinationURL) {
        errors.destination_url = "Destination URL must be present";
      }

      if (!hostStatusWebhookHostPercentage) {
        errors.host_percentage = "Host percentage must be present";
      }

      if (!hostStatusWebhookDaysCount) {
        errors.days_count = "Days count must be present";
      }
    }

    if (enableHostExpiry) {
      if (isNaN(hostExpiryWindow) || Number(hostExpiryWindow) <= 0) {
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

    if (Object.keys(errors).length !== 0) {
      return false;
    }

    // formDataToSubmit mirrors formatting of API not UI
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
        port: smtpPort,
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
        host_expiry_window: hostExpiryWindow,
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
    console.log("formDataToSubmit", formDataToSubmit);
    handleSubmit(formDataToSubmit);
  };

  // SECTIONS
  const renderOrganizationInfoSection = () => {
    return (
      <div className={`${baseClass}__section`}>
        <h2>
          <a id="organization-info">Organization info?</a>
        </h2>
        <div className={`${baseClass}__inputs`}>
          <InputField
            label="Organization name"
            onChange={handleInputChange}
            target
            name="orgName"
            value={orgName}
            error={formErrors.org_name}
          />
          <InputField
            label="Organization avatar URL"
            onChange={handleInputChange}
            name="orgLogoURL"
            target
            value={orgLogoURL}
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
            target
            name="serverurl"
            value={serverURL}
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
            target
            name="enableSSO"
            value={enableSSO}
          >
            Enable single sign on
          </Checkbox>
        </div>

        <div className={`${baseClass}__inputs`}>
          <InputField
            label="Identity provider name"
            onChange={handleInputChange}
            value={idpName}
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
            target
            name="entityID"
            value={entityID}
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
            target
            name="issuerURI"
            value={issuerURI}
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
            target
            name="idpImageURL"
            value={idpImageURL}
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
            target
            name="metadata"
            value={metadata}
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
            target
            name="metadataURL"
            value={metadataURL}
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
            target
            name="enableSSOIDPLogin"
            value={enableSSOIDPLogin}
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
            target
            name="smtpUsername"
            value={smtpUsername}
          />
          <InputField
            label="SMTP password"
            type="password"
            onChange={handleInputChange}
            target
            name="smtpPassword"
            value={smtpPassword}
          />
          <Dropdown
            label="Auth method"
            options={authMethodOptions}
            placeholder=""
            onChange={handleInputChange}
            target
            name="smtpAuthenticationMethod"
            value={smtpAuthenticationMethod}
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
                formData.smtp_settings.configured
                  ? "configured"
                  : "notconfigured"
              }`}
            >
              STATUS:{" "}
              <em>
                {formData.smtp_settings.configured
                  ? "CONFIGURED"
                  : "NOT CONFIGURED"}
              </em>
            </small>
          </a>
        </h2>
        <div className={`${baseClass}__inputs`}>
          <Checkbox
            onChange={handleInputChange}
            target
            name="enableSMTP"
            value={enableSMTP}
          >
            Enable SMTP
          </Checkbox>
        </div>

        <div className={`${baseClass}__inputs`}>
          <InputField
            label="Sender address"
            onChange={handleInputChange}
            target
            name="smtpSenderAddress"
            value={smtpSenderAddress}
          />
        </div>
        <div className={`${baseClass}__details`}>
          <IconToolTip text={"The sender address for emails from Fleet."} />
        </div>

        <div className={`${baseClass}__inputs ${baseClass}__inputs--smtp`}>
          <InputField
            label="SMTP server"
            onChange={handleInputChange}
            target
            name="smtpServer"
            value={smtpServer}
          />
          <InputField
            label="&nbsp;"
            type="number"
            onChange={handleInputChange}
            target
            name="smtpPort"
            value={smtpPort}
          />
          <Checkbox
            onChange={handleInputChange}
            target
            name="smtpEnableSSLTLS"
            value={smtpEnableSSLTLS}
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
            target
            name="smtpAuthenticationType"
            value={smtpAuthenticationType}
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
          <YamlAce
            onChange={handleInputChange}
            target
            name="agentOptions" // TODO
            value={agentOptions} // TODO
            error={formErrors.agent_options}
            wrapperClassName={`${baseClass}__text-editor-wrapper`}
          />
          {/* this might be tricky */}
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
            target
            name="enableHostStatusWebhook"
            value={enableHostStatusWebhook}
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
            target
            name="hostStatusWebhookDestinationURL"
            value={hostStatusWebhookDestinationURL}
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
            target
            name="hostStatusWebhookHostPercentage"
            value={hostStatusWebhookHostPercentage}
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
            target
            name="hostStatusWebhookDaysCount"
            value={hostStatusWebhookDaysCount}
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
            target
            name="enableUsageStatistics"
            value={enableUsageStatistics}
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
                  target
                  name="domain"
                  value={domain}
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
                  target
                  name="verifySSLCerts"
                  value={verifySSLCerts}
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
                  target
                  name="enableStartTLS"
                  value={enableStartTLS}
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
                  target
                  name="enableHostExpiry"
                  value={enableHostExpiry}
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
                  onChange={handleInputChange}
                  target
                  name="hostExpiryWindow"
                  value={hostExpiryWindow}
                  disabled={!enableHostExpiry}
                  label="Host Expiry Window"
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
                  target
                  name="disableLiveQuery"
                  value={disableLiveQuery}
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

    const json = {
      text:
        "More than X% of your hosts have not checked into Fleet for more than Y days. You’ve been sent this message because the Host status webhook is enabled in your Fleet instance.",
      data: {
        unseen_hosts: 1,
        total_hosts: 2,
        days_unseen: 3,
      },
    };

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
            <pre dangerouslySetInnerHTML={{ __html: syntaxHighlight(json) }} />
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

    const stats = {
      anonymousIdentifier: "9pnzNmrES3mQG66UQtd29cYTiX2+fZ4CYxDvh495720=",
      fleetVersion: "x.x.x",
      licenseTier: "free",
      numHostsEnrolled: 12345,
      numUsers: 12,
      numTeams: 3,
      numPolicies: 5,
      numLabels: 20,
      softwareInventoryEnabled: true,
      vulnDetectionEnabled: true,
      systemUsersEnabled: true,
      hostStatusWebhookEnabled: true,
    };

    return (
      <Modal
        title="Usage statistics"
        onExit={toggleUsageStatsPreviewModal}
        className={`${baseClass}__usage-stats-preview-modal`}
      >
        <>
          <p>An example JSON payload sent to Fleet Device Management Inc.</p>
          <pre dangerouslySetInnerHTML={{ __html: syntaxHighlight(stats) }} />
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
        <Button type="submit" variant="brand">
          Update settings
        </Button>
      </form>
      {renderUsageStatsPreviewModal()}
      {renderHostStatusWebhookPreviewModal()}
    </>
  );
};

export default AppConfigFormFunctional;
