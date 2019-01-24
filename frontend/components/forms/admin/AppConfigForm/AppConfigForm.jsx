import React, { Component } from 'react';
import PropTypes from 'prop-types';

import Button from 'components/buttons/Button';
import Checkbox from 'components/forms/fields/Checkbox';
import Dropdown from 'components/forms/fields/Dropdown';
import Form from 'components/forms/Form';
import formFieldInterface from 'interfaces/form_field';
import Icon from 'components/icons/Icon';
import InputField from 'components/forms/fields/InputField';
import OrgLogoIcon from 'components/icons/OrgLogoIcon';
import Slider from 'components/forms/fields/Slider';
import validate from 'components/forms/admin/AppConfigForm/validate';

const authMethodOptions = [
  { label: 'Plain', value: 'authmethod_plain' },
  { label: 'Cram MD5', value: 'authmethod_cram_md5' },
  { label: 'Login', value: 'authmethod_login' },
];
const authTypeOptions = [
  { label: 'Username and Password', value: 'authtype_username_password' },
  { label: 'None', value: 'authtype_none' },
];
const baseClass = 'app-config-form';
const formFields = [
  'authentication_method', 'authentication_type', 'domain', 'enable_ssl_tls', 'enable_start_tls',
  'kolide_server_url', 'org_logo_url', 'org_name', 'osquery_enroll_secret', 'password',
  'port', 'sender_address', 'server', 'user_name', 'verify_ssl_certs', 'idp_name', 'entity_id',
  'issuer_uri', 'idp_image_url', 'metadata', 'metadata_url', 'enable_sso', 'enable_smtp',
];
const Header = ({ showAdvancedOptions }) => {
  const CaratIcon = <Icon name={showAdvancedOptions ? 'downcarat' : 'upcarat'} />;

  return <span>Advanced Options {CaratIcon} <small>Most users do not need to modify these options.</small></span>;
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
      kolide_server_url: formFieldInterface.isRequired,
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
      enable_smtp: formFieldInterface.isRequired,
    }).isRequired,
    handleSubmit: PropTypes.func.isRequired,
    smtpConfigured: PropTypes.bool.isRequired,
  };

  constructor (props) {
    super(props);

    this.state = { revealSecret: false, showAdvancedOptions: false };
  }

  onToggleAdvancedOptions = (evt) => {
    evt.preventDefault();

    const { showAdvancedOptions } = this.state;

    this.setState({ showAdvancedOptions: !showAdvancedOptions });

    return false;
  }

  onToggleRevealSecret = (evt) => {
    evt.preventDefault();

    const { revealSecret } = this.state;

    this.setState({ revealSecret: !revealSecret });

    return false;
  }

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
          </div>
        </div>

        <div className={`${baseClass}__details`}>
          <p><strong>Domain</strong> - If you need to specify a HELO domain, you can do it here <em className="hint hint--brand">(Default: <strong>Blank</strong>)</em></p>
          <p><strong>Verify SSL Certs</strong> - Turn this off (not recommended) if you use a self-signed certificate <em className="hint hint--brand">(Default: <strong>On</strong>)</em></p>
          <p><strong>Enable STARTTLS</strong> - Detects if STARTTLS is enabled in your SMTP server and starts to use it. <em className="hint hint--brand">(Default: <strong>On</strong>)</em></p>
        </div>
      </div>
    );
  }

  renderSmtpSection = () => {
    const { fields } = this.props;

    if (fields.authentication_type.value === 'authtype_none') {
      return false;
    }

    return (
      <div className={`${baseClass}__smtp-section`}>
        <InputField
          {...fields.user_name}
          label="SMTP Username"
        />
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
  }

  render () {
    const { fields, handleSubmit, smtpConfigured } = this.props;
    const { onToggleAdvancedOptions, onToggleRevealSecret, renderAdvancedOptions, renderSmtpSection } = this;
    const { revealSecret, showAdvancedOptions } = this.state;

    return (
      <form className={baseClass} onSubmit={handleSubmit}>
        <div className={`${baseClass}__section`}>
          <h2>Organization Info</h2>
          <div className={`${baseClass}__inputs`}>
            <InputField
              {...fields.org_name}
              label="Organization Name"
            />
            <InputField
              {...fields.org_logo_url}
              label="Organization Avatar URL"
            />
          </div>
          <div className={`${baseClass}__details ${baseClass}__avatar-preview`}>
            <OrgLogoIcon src={fields.org_logo_url.value} />
            <p>Avatar Preview</p>
          </div>
        </div>
        <div className={`${baseClass}__section`}>
          <h2>Fleet Web Address</h2>
          <div className={`${baseClass}__inputs`}>
            <InputField
              {...fields.kolide_server_url}
              label="Fleet App URL"
              hint={<span>Include base path only (eg. no <code>/v1</code>)</span>}
            />
          </div>
          <div className={`${baseClass}__details`}>
            <p>What base URL should <strong>osqueryd</strong> clients use to connect and register with <strong>Fleet</strong>?</p>
            <p className={`${baseClass}__note`}><strong>Note:</strong> Please ensure the URL you choose is accessible to all endpoints that need to communicate with Fleet, otherwise they will not be able to correctly register.</p>
          </div>
        </div>

        <div className={`${baseClass}__section`}>
          <h2>SAML Single Sign On Options</h2>

          <div className={`${baseClass}__inputs`}>
            <Checkbox
              {...fields.enable_sso}
            >
              Enable Single Sign On
            </Checkbox>
          </div>

          <div className={`${baseClass}__inputs`}>
            <InputField
              {...fields.idp_name}
              label="Identity Provider Name"
            />
          </div>
          <div className={`${baseClass}__details`}>
            <p>A required human friendly name for the identity provider that will provide single sign on authentication.</p>
          </div>

          <div className={`${baseClass}__inputs`}>
            <InputField
              {...fields.entity_id}
              label="Entity ID"
              hint={<span>The URI you provide here must exactly match the Entity ID field used in identity provider configuration.</span>}
            />
          </div>
          <div className={`${baseClass}__details`}>
            <p>The required entity ID is a URI that you use to identify <strong>Fleet</strong> when configuring the identity provider.</p>
          </div>

          <div className={`${baseClass}__inputs`}>
            <InputField
              {...fields.issuer_uri}
              label="Issuer URI"
            />
          </div>
          <div className={`${baseClass}__details`}>
            <p>The issuer URI supplied by the identity provider.</p>
          </div>

          <div className={`${baseClass}__inputs`}>
            <InputField
              {...fields.idp_image_url}
              label="IDP Image URL"
            />
          </div>
          <div className={`${baseClass}__details`}>
            <p>An optional link to an image such as a logo for the identity provider.</p>
          </div>

          <div className={`${baseClass}__inputs`}>
            <InputField
              {...fields.metadata}
              label="Metadata"
              type="textarea"
            />
          </div>
          <div className={`${baseClass}__details`}>
            <p>Metadata provided by the identity provider. Either metadata or a metadata url must be provided.</p>
          </div>

          <div className={`${baseClass}__inputs`}>
            <InputField
              {...fields.metadata_url}
              label="Metadata URL"
              hint={<span>If available from the identity provider, this is the preferred means of providing metadata.</span>}
            />
          </div>
          <div className={`${baseClass}__details`}>
            <p>A URL that references the identity provider metadata.</p>
          </div>

        </div>

        <div className={`${baseClass}__section`}>
          <h2>SMTP Options <small className={`smtp-options smtp-options--${smtpConfigured ? 'configured' : 'notconfigured'}`}>STATUS: <em>{smtpConfigured ? 'CONFIGURED' : 'NOT CONFIGURED'}</em></small></h2>
          <div className={`${baseClass}__inputs`}>
            <Checkbox
              {...fields.enable_smtp}
            >
              Enable SMTP
            </Checkbox>
          </div>

          <div className={`${baseClass}__inputs`}>
            <InputField
              {...fields.sender_address}
              label="Sender Address"
            />
          </div>
          <div className={`${baseClass}__details`}>
            <p>The address email recipients will see all messages that are sent from the <strong>Fleet</strong> application.</p>
          </div>
          <div className={`${baseClass}__inputs ${baseClass}__inputs--smtp`}>
            <InputField
              {...fields.server}
              label="SMTP Server"
            />
            <InputField
              {...fields.port}
              label="&nbsp;"
              type="number"
            />
            <Checkbox
              {...fields.enable_ssl_tls}
            >
              Use SSL/TLS to connect (recommended)
            </Checkbox>
          </div>
          <div className={`${baseClass}__details`}>
            <p>The hostname / IP address and corresponding port of your organization&apos;s SMTP server.</p>
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
            <p>If your mail server requires authentication, you need to specify the authentication type here.</p>
            <p><strong>No Authentication</strong> - Select this if your SMTP is open.</p>
            <p><strong>Username & Password</strong> - Select this if your SMTP server requires authentication with a username and password.</p>
          </div>
        </div>
        <div className={`${baseClass}__section`}>
          <h2>Osquery Enrollment Secret</h2>
          <div className={`${baseClass}__inputs`}>
            <p className={`${baseClass}__enroll-secret-label`}>
              This is the secret that you use to enroll osquery agents with Fleet:
              <Button variant="unstyled" onClick={onToggleRevealSecret}>Reveal Secret</Button>
            </p>
            <InputField
              {...fields.osquery_enroll_secret}
              type={revealSecret ? 'input' : 'password'}
            />
          </div>
        </div>
        <div className={`${baseClass}__section`}>
          <h2><a href="#advancedOptions" onClick={onToggleAdvancedOptions} className={`${baseClass}__show-options`}><Header showAdvancedOptions={showAdvancedOptions} /></a></h2>
          {renderAdvancedOptions()}
        </div>
        <Button
          type="submit"
          variant="brand"
        >
          UPDATE SETTINGS
        </Button>
      </form>
    );
  }
}

export default Form(AppConfigForm, {
  fields: formFields,
  validate,
});
