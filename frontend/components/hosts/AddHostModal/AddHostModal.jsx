import React, { Component } from 'react';
import PropTypes from 'prop-types';
import FileSaver from 'file-saver';

import Kolide from 'kolide';
import Button from 'components/buttons/Button';
import configInterface from 'interfaces/config';
import enrollSecretInterface from 'interfaces/enroll_secret';
import EnrollSecretTable from 'components/config/EnrollSecretTable';
import Icon from 'components/icons/Icon';

const baseClass = 'add-host-modal';


class AddHostModal extends Component {
  static propTypes = {
    onReturnToApp: PropTypes.func,
    enrollSecret: enrollSecretInterface,
    config: configInterface,
  };

  constructor(props) {
    super(props);
    this.state = { fetchCertificateError: undefined };
  }

  componentDidMount() {
    Kolide.config.loadCertificate()
      .then((certificate) => {
        this.setState({ certificate });
      })
      .catch(() => {
        this.setState({ fetchCertificateError: 'Failed to load certificate. Is Fleet App URL configured properly?' });
      });
  }

  onFetchCertificate = (evt) => {
    evt.preventDefault();

    const { certificate } = this.state;

    const filename = 'fleet.pem';
    const file = new global.window.File([certificate], filename, { type: 'application/x-pem-file' });

    FileSaver.saveAs(file);

    return false;
  }

  render() {
    const {
      config,
      onReturnToApp,
      enrollSecret,
    } = this.props;

    const { fetchCertificateError } = this.state;

    let tlsHostname = config.kolide_server_url;
    try {
      const serverUrl = new URL(config.kolide_server_url);
      tlsHostname = serverUrl.hostname;
      if (serverUrl.port) {
        tlsHostname += `:${serverUrl.port}`;
      }
    } catch (e) {
      if (!(e instanceof TypeError)) {
        throw e;
      }
    }

    const flagfileContent = `--enroll_secret_path=secret.txt
--tls_server_certs=fleet.pem
--tls_hostname=${tlsHostname}
--host_identifier=uuid
--enroll_tls_endpoint=/api/v1/osquery/enroll
--config_plugin=tls
--config_tls_endpoint=/api/v1/osquery/config
--config_refresh=10
--disable_distributed=false
--distributed_plugin=tls
--distributed_interval=10
--distributed_tls_max_attempts=3
--distributed_tls_read_endpoint=/api/v1/osquery/distributed/read
--distributed_tls_write_endpoint=/api/v1/osquery/distributed/write
--logger_plugin=tls
--logger_tls_endpoint=/api/v1/osquery/log
--logger_tls_period=10
--disable_carver=false
--carver_start_endpoint=/api/v1/osquery/carve/begin
--carver_continue_endpoint=/api/v1/osquery/carve/block
--carver_block_size=2000000`;

    const onDownloadFlagfile = (evt) => {
      evt.preventDefault();

      const filename = 'flagfile.txt';
      const file = new global.window.File([flagfileContent], filename);

      FileSaver.saveAs(file);

      return false;
    };

    return (
      <div className={baseClass}>
        <div className={`${baseClass}__manual-install-content`}>
          <div className={`${baseClass}__documentation-link`}>
            <h4>
              <a
                href="https://github.com/kolide/fleet/blob/master/docs/infrastructure/adding-hosts-to-fleet.md"
                target="_blank"
                rel="noopener noreferrer"
              >
                Add Hosts Documentation <Icon name="external-link" />
              </a>
            </h4>
          </div>
          <ol className={`${baseClass}__install-steps`}>
            <li>
              <h4>1. Enroll Secret</h4>
              <p>
                Provide an active enroll secret to allow osquery to authenticate with the Fleet server:
              </p>
              <div className={`${baseClass}__secret-wrapper`}>
                <EnrollSecretTable secrets={enrollSecret} />
              </div>
            </li>
            <li>
              <h4>2. Server Certificate</h4>
              <p>
                Provide the TLS certificate used by the Fleet server to enable secure connections from osquery:
              </p>
              <p>
                { fetchCertificateError
                  ? <span className={`${baseClass}__error`}>{fetchCertificateError}</span>
                  : <a href="#downloadCertificate" onClick={this.onFetchCertificate}>Download Certificate</a>
                }
              </p>
            </li>
            <li>
              <h4>3. Flagfile</h4>
              <p>
                If using the enroll secret and server certificate downloaded above, use the generated flagfile. In some configurations, modifications may need to be made:
              </p>
              <p>
                <a href="#downloadFlagfile" onClick={onDownloadFlagfile}>Download Flagfile</a>
              </p>
            </li>
            <li>
              <h4>4. Run Osquery</h4>
              <p>
                Run osquery from the directory containing the above files (may require sudo or Run as Administrator privileges):
              </p>
              <pre>osqueryd --flagfile=flagfile.txt --verbose</pre>
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
