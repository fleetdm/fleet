import React, { Component } from 'react';
import PropTypes from 'prop-types';

import Button from 'components/buttons/Button';
import Icon from 'components/icons/Icon';
import InputField from 'components/forms/fields/InputField';
import { renderFlash } from 'redux/nodes/notifications/actions';
import {
  copyText,
  COPY_TEXT_SUCCESS,
  COPY_TEXT_ERROR,
} from 'utilities/copy_text';
import certificate from '../../../../assets/images/osquery-certificate.svg';

const baseClass = 'add-host-modal';

class AddHostModal extends Component {
  static propTypes = {
    dispatch: PropTypes.func,
    onFetchCertificate: PropTypes.func,
    onReturnToApp: PropTypes.func,
    osqueryEnrollSecret: PropTypes.string,
  };

  constructor(props) {
    super(props);

    this.state = { revealSecret: false };
  }

  onCopySecret = (elementClass) => {
    return (evt) => {
      evt.preventDefault();

      const { dispatch } = this.props;

      if (copyText(elementClass)) {
        dispatch(renderFlash('success', COPY_TEXT_SUCCESS));
      } else {
        this.setState({ revealSecret: true });
        dispatch(renderFlash('error', COPY_TEXT_ERROR));
      }
    };
  };

  toggleSecret = (evt) => {
    const { revealSecret } = this.state;
    evt.preventDefault();

    this.setState({ revealSecret: !revealSecret });
    return false;
  };

  render() {
    const { onCopySecret, toggleSecret } = this;
    const { revealSecret } = this.state;
    const {
      onFetchCertificate,
      onReturnToApp,
      osqueryEnrollSecret,
    } = this.props;

    return (
      <div className={baseClass}>
        <p>
          Follow the instructions below to add hosts to your Fleet Instance.
        </p>

        <div className={`${baseClass}__manual-install-header`}>
          <Icon name="wrench-hand" />
          <h2>Manual Install</h2>
          <h3>
            Fully Customize Your <strong>Osquery</strong> Installation
          </h3>
        </div>

        <div className={`${baseClass}__manual-install-content`}>
          <ol className={`${baseClass}__install-steps`}>
            <li>
              <h4>
                <a
                  href="https://github.com/kolide/fleet/blob/master/docs/infrastructure/adding-hosts-to-fleet.md"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  Fleet / Osquery - Install Docs <Icon name="external-link" />
                </a>
              </h4>
              <p>
                In order to install <strong>osquery</strong> on a client you
                will need the following information:
              </p>
            </li>
            <li>
              <h4>Retrieve Osquery Enroll Secret</h4>
              <p>
                The following is your enroll secret:
                <a
                  href="#revealSecret"
                  onClick={toggleSecret}
                  className={`${baseClass}__reveal-secret`}
                >
                  {revealSecret ? 'Hide' : 'Reveal'} Secret
                </a>
              </p>
              <div className={`${baseClass}__secret-wrapper`}>
                <InputField
                  disabled
                  inputWrapperClass={`${baseClass}__secret-input`}
                  name="osqueryd-secret"
                  type={revealSecret ? 'text' : 'password'}
                  value={osqueryEnrollSecret}
                />
                <Button
                  variant="unstyled"
                  className={`${baseClass}__secret-copy-icon`}
                  onClick={onCopySecret(`.${baseClass}__secret-input`)}
                >
                  <Icon name="clipboard" />
                </Button>
              </div>
            </li>
            <li>
              <h4>Download Server Certificate (Optional)</h4>
              <p>
                If you use the native osquery TLS plugins, Osquery requires the
                same TLS certificate that Fleet is using in order to
                authenticate. You can fetch the certificate below:
              </p>
              <p className={`${baseClass}__download-cert`}>
                <Button variant="unstyled" onClick={onFetchCertificate}>
                  <img src={certificate} role="presentation" />
                  <span>Fetch Fleet Certificate</span>
                </Button>
              </p>
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
