import React, { Component, PropTypes } from 'react';

import Button from 'components/buttons/Button';
import Icon from 'components/icons/Icon';
import InputField from 'components/forms/fields/InputField';
import { renderFlash } from 'redux/nodes/notifications/actions';
import { copyText } from './helpers';
import certificate from '../../../../assets/images/osquery-certificate.svg';

const baseClass = 'add-host-modal';

class AddHostModal extends Component {
  static propTypes = {
    dispatch: PropTypes.func,
    onReturnToApp: PropTypes.func,
  };

  constructor (props) {
    super(props);

    this.state = {
      revealSecret: false,
      secretText: '1234567890',
    };
  }

  onCopySecret = (elementClass) => {
    return (evt) => {
      evt.preventDefault();

      const { dispatch } = this.props;

      if (copyText(elementClass)) {
        dispatch(renderFlash('success', 'Text copied to clipboard'));
      } else {
        this.setState({ revealSecret: true });
        dispatch(renderFlash('error', 'Text not copied. Use CMD + C to copy text'));
      }
    };
  }

  onDownloadCertificate = (evt) => {
    evt.preventDefault();

    console.log('Download the Certificate');

    return false;
  }

  toggleSecret = (evt) => {
    const { revealSecret } = this.state;
    evt.preventDefault();

    this.setState({ revealSecret: !revealSecret });
    return false;
  }

  render () {
    const { onCopySecret, onDownloadCertificate, toggleSecret } = this;
    const { revealSecret } = this.state;
    const { onReturnToApp } = this.props;

    return (
      <div className={baseClass}>
        <p>Follow the instructions below to add hosts to your Kolide Instance.</p>

        <div className={`${baseClass}__manual-install-header`}>
          <Icon name="wrench-hand" />
          <h2>Manual Install</h2>
          <h3>Fully Customize Your <strong>Osquery</strong> Installation</h3>
        </div>

        <div className={`${baseClass}__manual-install-content`}>
          <ol className={`${baseClass}__install-steps`}>
            <li>
              <h4><a href="https://docs.kolide.co/using-kolide/master/hosts/adding-hosts.html" target="_blank" rel="noopener noreferrer">Kolide / Osquery - Install Docs <Icon name="external-link" /></a></h4>
              <p>In order to install <strong>osquery</strong> on a client you will need the items below:</p>
            </li>
            <li>
              <h4>Download Osquery Package and Certificate</h4>
              <p>Osquery requires the same TLS certificate that Kolide is using in order to authenticate. You can fetch the certificate below:</p>
              <p className={`${baseClass}__download-cert`}>
                <a href="#downloadCertificate" onClick={onDownloadCertificate}>
                  <img src={certificate} role="presentation" />
                  <span>Fetch Kolide Certificate</span>
                </a>
              </p>
            </li>
            <li>
              <h4>Retrieve Osquery Enroll Secret</h4>
              <p>
                When prompted, enter the provided secret code into <strong>osqueryd</strong>:
                <a href="#revealSecret" onClick={toggleSecret} className={`${baseClass}__reveal-secret`}>{revealSecret ? 'Hide' : 'Reveal'} Secret</a>
              </p>
              <div className={`${baseClass}__secret-wrapper`}>
                <InputField
                  disabled
                  inputWrapperClass={`${baseClass}__secret-input`}
                  name="osqueryd-secret"
                  type={revealSecret ? 'text' : 'password'}
                  value={this.state.secretText}
                />
                <Button variant="unstyled" className={`${baseClass}__secret-copy-icon`} onClick={onCopySecret(`.${baseClass}__secret-input`)}>
                  <Icon name="clipboard" />
                </Button>
              </div>

            </li>
          </ol>
        </div>

        <div className={`${baseClass}__button-wrap`}>
          <Button onClick={onReturnToApp} variant="success">
            Return To App
          </Button>
        </div>
      </div>
    );
  }
}

export default AddHostModal;
