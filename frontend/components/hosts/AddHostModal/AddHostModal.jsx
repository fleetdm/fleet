import React, { Component } from 'react';
import PropTypes from 'prop-types';

import Button from 'components/buttons/Button';
import enrollSecretInterface from 'interfaces/enroll_secret';
import EnrollSecretTable from 'components/config/EnrollSecretTable';
import Icon from 'components/icons/Icon';
import certificate from '../../../../assets/images/osquery-certificate.svg';

const baseClass = 'add-host-modal';

class AddHostModal extends Component {
  static propTypes = {
    onFetchCertificate: PropTypes.func,
    onReturnToApp: PropTypes.func,
    enrollSecret: enrollSecretInterface,
  };

  render() {
    const {
      onFetchCertificate,
      onReturnToApp,
      enrollSecret,
    } = this.props;

    return (
      <div className={baseClass}>
        <p>
          Follow the instructions below to add hosts to your Fleet Instance.
        </p>

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
            </li>
            <li>
              <h4>Osquery Enroll Secret</h4>
              <p>
                Provide osquery with one of the following active enroll secrets:
              </p>
              <div className={`${baseClass}__secret-wrapper`}>
                <EnrollSecretTable secrets={enrollSecret} />
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
                  <img src={certificate} alt="" />
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
