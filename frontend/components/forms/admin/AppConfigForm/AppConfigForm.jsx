import React, { Component } from "react";
import PropTypes from "prop-types";
import { syntaxHighlight } from "fleet/helpers";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import Dropdown from "components/forms/fields/Dropdown";
import Form from "components/forms/Form";
import formFieldInterface from "interfaces/form_field";
import enrollSecretInterface from "interfaces/enroll_secret";
import EnrollSecretTable from "components/config/EnrollSecretTable";
import InputField from "components/forms/fields/InputField";
import OrgLogoIcon from "components/icons/OrgLogoIcon";
import Slider from "components/forms/fields/Slider";
import validate from "components/forms/admin/AppConfigForm/validate";
import IconToolTip from "components/IconToolTip";
import InfoBanner from "components/InfoBanner/InfoBanner";
import YamlAce from "components/YamlAce";
import Modal from "components/modals/Modal";
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
  "enable_analytics",
];
const Header = ({ showAdvancedOptions }) => {
  const CaratIcon = (
    <Button
      className={`button button--unstyled ${
        showAdvancedOptions ? "upcarat" : "downcarat"
      }`}
    />
  );

  return (
    <span>
      Advanced Options {CaratIcon}{" "}
      <small>Most users do not need to modify these options.</small>
    </span>
  );
};

Header.propTypes = { showAdvancedOptions: PropTypes.bool.isRequired };

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
      enable_analytics: formFieldInterface.isRequired,
    }).isRequired,
    enrollSecret: enrollSecretInterface.isRequired,
    handleSubmit: PropTypes.func.isRequired,
    smtpConfigured: PropTypes.bool.isRequired,
  };

  constructor(props) {
    super(props);

    this.state = {
      showAdvancedOptions: false,
      showUsageStatsPreviewModal: false,
    };
  }

  onToggleAdvancedOptions = (evt) => {
    evt.preventDefault();

    const { showAdvancedOptions } = this.state;

    this.setState({ showAdvancedOptions: !showAdvancedOptions });

    return false;
  };

  toggleUsageStatsPreviewModal = () => {
    const { showUsageStatsPreviewModal } = this.state;
    this.setState({
      showUsageStatsPreviewModal: !showUsageStatsPreviewModal,
    });
  };

  renderAdvancedOptions = () => {
    const { fields } = this.props;
    const { showAdvancedOptions } = this.state;

    if (!showAdvancedOptions) {
      return false;
    }

    return (
      <div>
        <div className={`${baseClass}__inputs`}>
          <div className={`${baseClass}__smtp-section`}>
            <InputField {...fields.domain} label="Domain" />
            <Slider {...fields.verify_ssl_certs} label="Verify SSL Certs?" />
            <Slider {...fields.enable_start_tls} label="Enable STARTTLS?" />
            <Slider {...fields.host_expiry_enabled} label="Host Expiry" />
            <InputField
              {...fields.host_expiry_window}
              disabled={!fields.host_expiry_enabled.value}
              label="Host Expiry Window"
            />
            <Slider
              {...fields.live_query_disabled}
              label="Disable Live Queries?"
            />
          </div>
        </div>

        <div className={`${baseClass}__details`}>
          <IconToolTip
            isHtml
            text={
              '\
              <p><strong>Domain</strong> - If you need to specify a HELO domain, you can do it here <em className="hint hint--brand">(Default: <strong>Blank</strong>)</em></p>\
              <p><strong>Verify SSL Certs</strong> - Turn this off (not recommended) if you use a self-signed certificate <em className="hint hint--brand">(Default: <strong>On</strong>)</em></p>\
              <p><strong>Enable STARTTLS</strong> - Detects if STARTTLS is enabled in your SMTP server and starts to use it. <em className="hint hint--brand">(Default: <strong>On</strong>)</em></p>\
              <p><strong>Host Expiry</strong> - When enabled, allows automatic cleanup of hosts that have not communicated with Fleet in some number of days. <em className="hint hint--brand">(Default: <strong>Off</strong>)</em></p>\
              <p><strong>Host Expiry Window</strong> - If a host has not communicated with Fleet in the specified number of days, it will be removed.</p>\
              <p><strong>Disable Live Queries</strong> - When enabled, disables the ability to run live queries (ad hoc queries executed via the UI or fleetctl). <em className="hint hint--brand">(Default: <strong>Off</strong>)</em></p>\
            '
            }
          />
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
        <InputField {...fields.user_name} label="SMTP Username" />
        <InputField
          {...fields.password}
          label="SMTP Password"
          type="password"
        />
        <Dropdown
          {...fields.authentication_method}
          label="Auth Method"
          options={authMethodOptions}
          placeholder=""
        />
      </div>
    );
  };

  renderUsageStatsPreviewModal = () => {
    const { toggleUsageStatsPreviewModal } = this;
    const { showUsageStatsPreviewModal } = this.state;

    if (!showUsageStatsPreviewModal) {
      return null;
    }

    const json = {
      anonymous_identifier: "wmTH972f06USpahr41LHpgLKAhgZL",
      fleet_version: "x.x.x",
      hosts_enrolled_count: 12345,
    };

    return (
      <Modal
        title="Usage statistics"
        onExit={toggleUsageStatsPreviewModal}
        className={`${baseClass}__usage-stats-preview-modal`}
      >
        <p>An example JSON payload sent to Fleet Device Management Inc.</p>
        <pre dangerouslySetInnerHTML={{ __html: syntaxHighlight(json) }} />
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
      onToggleAdvancedOptions,
      renderAdvancedOptions,
      renderSmtpSection,
      toggleUsageStatsPreviewModal,
      renderUsageStatsPreviewModal,
    } = this;
    const { showAdvancedOptions } = this.state;

    return (
      <>
        <form className={baseClass} onSubmit={handleSubmit}>
          <div className={`${baseClass}__section`}>
            <h2>
              <a id="organization-info">Organization Info</a>
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
                label="Fleet App URL"
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
              <a id="saml">SAML Single Sign On Options</a>
            </h2>

            <div className={`${baseClass}__inputs`}>
              <Checkbox {...fields.enable_sso}>Enable Single Sign On</Checkbox>
            </div>

            <div className={`${baseClass}__inputs`}>
              <InputField {...fields.idp_name} label="Identity Provider Name" />
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
              <InputField {...fields.idp_image_url} label="IDP Image URL" />
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
                SMTP Options{" "}
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
              <InputField {...fields.sender_address} label="Sender Address" />
            </div>
            <div className={`${baseClass}__details`}>
              <IconToolTip text={"The sender address for emails from Fleet."} />
            </div>

            <div className={`${baseClass}__inputs ${baseClass}__inputs--smtp`}>
              <InputField {...fields.server} label="SMTP Server" />
              <InputField {...fields.port} label="&nbsp;" type="number" />
              <Checkbox {...fields.enable_ssl_tls}>
                Use SSL/TLS to connect (recommended)
              </Checkbox>
            </div>
            <div className={`${baseClass}__details`}>
              <IconToolTip
                text={
                  "The hostname / IP address and corresponding port of your organization&apos;s SMTP server."
                }
              />
            </div>

            <div className={`${baseClass}__inputs`}>
              <Dropdown
                {...fields.authentication_type}
                label="Authentication Type"
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
              <a id="osquery-enrollment-secrets">Osquery Enrollment Secrets</a>
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
              <a
                id="advanced-options"
                onClick={onToggleAdvancedOptions}
                className={`${baseClass}__show-options`}
              >
                <Header showAdvancedOptions={showAdvancedOptions} />
              </a>
            </h2>
            {renderAdvancedOptions()}
          </div>
          <Button type="submit" variant="brand">
            Update settings
          </Button>
        </form>
        {renderUsageStatsPreviewModal()}
      </>
    );
  }
}

export default Form(AppConfigForm, {
  fields: formFields,
  validate,
});
