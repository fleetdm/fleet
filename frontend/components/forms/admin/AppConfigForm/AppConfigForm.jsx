import React, { Component } from "react";
import PropTypes from "prop-types";
import { syntaxHighlight } from "fleet/helpers";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import Dropdown from "components/forms/fields/Dropdown";
import Form from "components/forms/Form";
import formFieldInterface from "interfaces/form_field";
import enrollSecretInterface from "interfaces/enroll_secret";
import EnrollSecretTable from "components/EnrollSecretTable";
import InputField from "components/forms/fields/InputField";
import OrgLogoIcon from "components/icons/OrgLogoIcon";
import validate from "components/forms/admin/AppConfigForm/validate";
import IconToolTip from "components/IconToolTip";
import InfoBanner from "components/InfoBanner/InfoBanner";
import YamlAce from "components/YamlAce";
import Modal from "components/Modal";
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

const baseClass = "app-config-form";
const formFields = [
  "authentication_method",
  "authentication_type",
  "domain",
  "enable_ssl_tls",
  "enable_start_tls",
  "server_url",
  "org_logo_url",
  "org_name",
  "osquery_enroll_secret",
  "password",
  "port",
  "sender_address",
  "server",
  "user_name",
  "verify_ssl_certs",
  "idp_name",
  "entity_id",
  "issuer_uri",
  "idp_image_url",
  "metadata",
  "metadata_url",
  "enable_sso",
  "enable_sso_idp_login",
  "enable_smtp",
  "host_expiry_enabled",
  "host_expiry_window",
  "live_query_disabled",
  "agent_options",
  "enable_host_status_webhook",
  "destination_url",
  "host_percentage",
  "days_count",
  "enable_analytics",
];
class AppConfigForm extends Component {
  static propTypes = {
    fields: PropTypes.shape({
      authentication_method: formFieldInterface.isRequired,
      authentication_type: formFieldInterface.isRequired,
      domain: formFieldInterface.isRequired,
      enable_ssl_tls: formFieldInterface.isRequired,
      enable_start_tls: formFieldInterface.isRequired,
      server_url: formFieldInterface.isRequired,
      org_logo_url: formFieldInterface.isRequired,
      org_name: formFieldInterface.isRequired,
      password: formFieldInterface.isRequired,
      port: formFieldInterface.isRequired,
      sender_address: formFieldInterface.isRequired,
      server: formFieldInterface.isRequired,
      user_name: formFieldInterface.isRequired,
      verify_ssl_certs: formFieldInterface.isRequired,
      entity_id: formFieldInterface.isRequired,
      issuer_uri: formFieldInterface.isRequired,
      idp_image_url: formFieldInterface.isRequired,
      metadata: formFieldInterface.isRequired,
      metadata_url: formFieldInterface.isRequired,
      idp_name: formFieldInterface.isRequired,
      enable_sso: formFieldInterface.isRequired,
      enable_sso_idp_login: formFieldInterface.isRequired,
      enable_smtp: formFieldInterface.isRequired,
      host_expiry_enabled: formFieldInterface.isRequired,
      host_expiry_window: formFieldInterface.isRequired,
      live_query_disabled: formFieldInterface.isRequired,
      agent_options: formFieldInterface.isRequired,
      enable_host_status_webhook: formFieldInterface.isRequired,
      destination_url: formFieldInterface,
      host_percentage: formFieldInterface,
      days_count: formFieldInterface,
      enable_analytics: formFieldInterface.isRequired,
    }).isRequired,
    enrollSecret: PropTypes.arrayOf(enrollSecretInterface).isRequired,
    handleSubmit: PropTypes.func.isRequired,
    smtpConfigured: PropTypes.bool.isRequired,
  };

  constructor(props) {
    super(props);

    this.state = {
      showHostStatusWebhookPreviewModal: false,
      showUsageStatsPreviewModal: false,
    };
  }

  toggleHostStatusWebhookPreviewModal = () => {
    const { showHostStatusWebhookPreviewModal } = this.state;
    this.setState({
      showHostStatusWebhookPreviewModal: !showHostStatusWebhookPreviewModal,
    });
  };

  toggleUsageStatsPreviewModal = () => {
    const { showUsageStatsPreviewModal } = this.state;
    this.setState({
      showUsageStatsPreviewModal: !showUsageStatsPreviewModal,
    });
  };

  renderAdvancedOptions = () => {
    const { fields } = this.props;

    return (
      <div className={`${baseClass}__advanced-options`}>
        <p className={`${baseClass}__section-description`}>
          Most users do not need to modify these options.
        </p>
        <div className={`${baseClass}__inputs`}>
          <div className={`${baseClass}__form-fields`}>
            <div className="tooltip-wrap tooltip-wrap--input">
              <InputField {...fields.domain} label="Domain" />
              <IconToolTip
                isHtml
                text={
                  '<p>If you need to specify a HELO domain, <br />you can do it here <em className="hint hint--brand">(Default: <strong>Blank</strong>)</em></p>'
                }
              />
            </div>
            <div className="tooltip-wrap">
              <Checkbox {...fields.verify_ssl_certs}>Verify SSL certs</Checkbox>
              <IconToolTip
                isHtml
                text={
                  '<p>Turn this off (not recommended) <br />if you use a self-signed certificate <em className="hint hint--brand"><br />(Default: <strong>On</strong>)</em></p>'
                }
              />
            </div>
            <div className="tooltip-wrap">
              <Checkbox {...fields.enable_start_tls}>Enable STARTTLS</Checkbox>
              <IconToolTip
                isHtml
                text={
                  '<p>Detects if STARTTLS is enabled <br />in your SMTP server and starts <br />to use it. <em className="hint hint--brand">(Default: <strong>On</strong>)</em></p>'
                }
              />
            </div>
            <div className="tooltip-wrap">
              <Checkbox {...fields.host_expiry_enabled}>Host expiry</Checkbox>
              <IconToolTip
                isHtml
                text={
                  '<p>When enabled, allows automatic cleanup <br />of hosts that have not communicated with Fleet <br />in some number of days. <em className="hint hint--brand">(Default: <strong>Off</strong>)</em></p>'
                }
              />
            </div>
            <div className="tooltip-wrap tooltip-wrap--input">
              <InputField
                {...fields.host_expiry_window}
                disabled={!fields.host_expiry_enabled.value}
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
              <Checkbox {...fields.live_query_disabled}>
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
    );
  };

  renderSmtpSection = () => {
    const { fields } = this.props;

    if (fields.authentication_type.value === "authtype_none") {
      return false;
    }

    return (
      <div className={`${baseClass}__smtp-section`}>
        <InputField {...fields.user_name} label="SMTP username" />
        <InputField
          {...fields.password}
          label="SMTP password"
          type="password"
        />
        <Dropdown
          {...fields.authentication_method}
          label="Auth method"
          options={authMethodOptions}
          placeholder=""
        />
      </div>
    );
  };

  renderHostStatusWebhookPreviewModal = () => {
    const { toggleHostStatusWebhookPreviewModal } = this;
    const { showHostStatusWebhookPreviewModal } = this.state;

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
      </Modal>
    );
  };

  renderUsageStatsPreviewModal = () => {
    const { toggleUsageStatsPreviewModal } = this;
    const { showUsageStatsPreviewModal } = this.state;

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
        <p>An example JSON payload sent to Fleet Device Management Inc.</p>
        <pre dangerouslySetInnerHTML={{ __html: syntaxHighlight(stats) }} />
        <div className="flex-end">
          <Button type="button" onClick={toggleUsageStatsPreviewModal}>
            Done
          </Button>
        </div>
      </Modal>
    );
  };

  render() {
    const { fields, handleSubmit, smtpConfigured, enrollSecret } = this.props;
    const {
      renderAdvancedOptions,
      renderSmtpSection,
      toggleHostStatusWebhookPreviewModal,
      toggleUsageStatsPreviewModal,
      renderHostStatusWebhookPreviewModal,
      renderUsageStatsPreviewModal,
    } = this;

    return (
      <>
        <form className={baseClass} onSubmit={handleSubmit} autoComplete="off">
          <div className={`${baseClass}__section`}>
            <h2>
              <a id="organization-info">Organization info</a>
            </h2>
            <div className={`${baseClass}__inputs`}>
              <InputField {...fields.org_name} label="Organization name" />
              <InputField
                {...fields.org_logo_url}
                label="Organization avatar URL"
              />
            </div>
            <div
              className={`${baseClass}__details ${baseClass}__avatar-preview`}
            >
              <OrgLogoIcon src={fields.org_logo_url.value} />
            </div>
          </div>
          <div className={`${baseClass}__section`}>
            <h2>
              <a id="fleet-web-address">Fleet web address</a>
            </h2>
            <div className={`${baseClass}__inputs`}>
              <InputField
                {...fields.server_url}
                label="Fleet app URL"
                hint={
                  <span>
                    Include base path only (eg. no <code>/v1</code>)
                  </span>
                }
              />
            </div>
            <div className={`${baseClass}__details`}>
              <IconToolTip
                text={"The base URL of this instance for use in Fleet links."}
              />
            </div>
          </div>

          <div className={`${baseClass}__section`}>
            <h2>
              <a id="saml">SAML single sign on options</a>
            </h2>

            <div className={`${baseClass}__inputs`}>
              <Checkbox {...fields.enable_sso}>Enable single sign on</Checkbox>
            </div>

            <div className={`${baseClass}__inputs`}>
              <InputField {...fields.idp_name} label="Identity provider name" />
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
                {...fields.entity_id}
                label="Entity ID"
                hint={
                  <span>
                    The URI you provide here must exactly match the Entity ID
                    field used in identity provider configuration.
                  </span>
                }
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
              <InputField {...fields.issuer_uri} label="Issuer URI" />
            </div>
            <div className={`${baseClass}__details`}>
              <IconToolTip
                text={"The issuer URI supplied by the identity provider."}
              />
            </div>

            <div className={`${baseClass}__inputs`}>
              <InputField {...fields.idp_image_url} label="IDP image URL" />
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
                {...fields.metadata}
                label="Metadata"
                type="textarea"
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
                {...fields.metadata_url}
                label="Metadata URL"
                hint={
                  <span>
                    If available from the identity provider, this is the
                    preferred means of providing metadata.
                  </span>
                }
              />
            </div>
            <div className={`${baseClass}__details`}>
              <IconToolTip
                text={"A URL that references the identity provider metadata."}
              />
            </div>

            <div className={`${baseClass}__inputs`}>
              <Checkbox {...fields.enable_sso_idp_login}>
                Allow SSO login initiated by Identity Provider
              </Checkbox>
            </div>
          </div>

          <div className={`${baseClass}__section`}>
            <h2>
              <a id="smtp">
                SMTP options{" "}
                <small
                  className={`smtp-options smtp-options--${
                    smtpConfigured ? "configured" : "notconfigured"
                  }`}
                >
                  STATUS:{" "}
                  <em>{smtpConfigured ? "CONFIGURED" : "NOT CONFIGURED"}</em>
                </small>
              </a>
            </h2>
            <div className={`${baseClass}__inputs`}>
              <Checkbox {...fields.enable_smtp}>Enable SMTP</Checkbox>
            </div>

            <div className={`${baseClass}__inputs`}>
              <InputField {...fields.sender_address} label="Sender address" />
            </div>
            <div className={`${baseClass}__details`}>
              <IconToolTip text={"The sender address for emails from Fleet."} />
            </div>

            <div className={`${baseClass}__inputs ${baseClass}__inputs--smtp`}>
              <InputField {...fields.server} label="SMTP server" />
              <InputField {...fields.port} label="&nbsp;" type="number" />
              <Checkbox {...fields.enable_ssl_tls}>
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
                {...fields.authentication_type}
                label="Authentication type"
                options={authTypeOptions}
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

          <div className={`${baseClass}__section`}>
            <h2>
              <a id="agent-options">Global agent options</a>
            </h2>
            <div className={`${baseClass}__yaml`}>
              <p className={`${baseClass}__section-description`}>
                This code will be used by osquery when it checks for
                configuration options.
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
                  <img
                    className="icon"
                    src={OpenNewTabIcon}
                    alt="open new tab"
                  />
                </a>
              </InfoBanner>
              <p className={`${baseClass}__component-label`}>
                <b>YAML</b>
              </p>
              <YamlAce
                {...fields.agent_options}
                error={fields.agent_options.error}
                wrapperClassName={`${baseClass}__text-editor-wrapper`}
              />
            </div>
          </div>

          <div className={`${baseClass}__section`}>
            <h2>
              <a id="host-status-webhook">Host status webhook</a>
            </h2>
            <div className={`${baseClass}__host-status-webhook`}>
              <p className={`${baseClass}__section-description`}>
                Send an alert if a portion of your hosts go offline.
              </p>
              <Checkbox {...fields.enable_host_status_webhook}>
                Enable host status webhook
              </Checkbox>
              <p className={`${baseClass}__section-description`}>
                A request will be sent to your configured <b>Destination URL</b>{" "}
                if the configured <b>Percentage of hosts</b> have not checked
                into Fleet for the configured <b>Number of days</b>.
              </p>
            </div>
            <div
              className={`${baseClass}__inputs ${baseClass}__inputs--webhook`}
            >
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
                {...fields.destination_url}
                placeholder="https://server.com/example"
                label="Destination URL"
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
            <div
              className={`${baseClass}__inputs ${baseClass}__host-percentage`}
            >
              <Dropdown
                {...fields.host_percentage}
                label="Percentage of hosts"
                options={percentageOfHosts}
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
                {...fields.days_count}
                label="Number of days"
                options={numberOfDays}
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

          <div className={`${baseClass}__section`}>
            <h2>
              <a id="usage-stats">Usage statistics</a>
            </h2>
            <p className={`${baseClass}__section-description`}>
              Help improve Fleet by sending anonymous usage statistics.
              <br />
              <br />
              This information helps our team better understand feature adoption
              and usage, and allows us to see how Fleet is adding value, so that
              we can make better product decisions.
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
              <Checkbox {...fields.enable_analytics}>
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

          <div className={`${baseClass}__section`}>
            <h2>
              <a id="advanced-options">Advanced options</a>
            </h2>
            {renderAdvancedOptions()}
          </div>
          <Button type="submit" variant="brand">
            Update settings
          </Button>
        </form>
        {renderUsageStatsPreviewModal()}
        {renderHostStatusWebhookPreviewModal()}
      </>
    );
  }
}

export default Form(AppConfigForm, {
  fields: formFields,
  validate,
});
