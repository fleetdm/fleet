import React, { Component } from "react";
import PropTypes from "prop-types";
import FileSaver from "file-saver";

import Kolide from "kolide";
import Button from "components/buttons/Button";
import configInterface from "interfaces/config";
import enrollSecretInterface from "interfaces/enroll_secret";
import EnrollSecretTable from "components/config/EnrollSecretTable";
import KolideIcon from "components/icons/KolideIcon";
import DownloadIcon from "../../../../assets/images/icon-download-12x12@2x.png";

const baseClass = "add-host-modal";

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
    Kolide.config
      .loadCertificate()
      .then((certificate) => {
        this.setState({ certificate });
      })
      .catch(() => {
        this.setState({
          fetchCertificateError:
            "Failed to load certificate. Is Fleet App URL configured properly?",
        });
      });
  }

  onFetchCertificate = (evt) => {
    evt.preventDefault();

    const { certificate } = this.state;

    const filename = "fleet.pem";
    const file = new global.window.File([certificate], filename, {
      type: "application/x-pem-file",
    });

    FileSaver.saveAs(file);

    return false;
  };

  render() {
    const { config, onReturnToApp, enrollSecret } = this.props;

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

    const flagfileContent = `# Server
--tls_hostname=${tlsHostname}
--tls_server_certs=fleet.pem

# Enrollment
--host_identifier=instance
--enroll_secret_path=secret.txt
--enroll_tls_endpoint=/api/v1/osquery/enroll

# Configuration
--config_plugin=tls
--config_tls_endpoint=/api/v1/osquery/config
--config_refresh=10

# Live query
--disable_distributed=false
--distributed_plugin=tls
--distributed_interval=10
--distributed_tls_max_attempts=3
--distributed_tls_read_endpoint=/api/v1/osquery/distributed/read
--distributed_tls_write_endpoint=/api/v1/osquery/distributed/write

# Logging
--logger_plugin=tls
--logger_tls_endpoint=/api/v1/osquery/log
--logger_tls_period=10

# File carving
--disable_carver=false
--carver_start_endpoint=/api/v1/osquery/carve/begin
--carver_continue_endpoint=/api/v1/osquery/carve/block
--carver_block_size=2000000`;

    const onDownloadFlagfile = (evt) => {
      evt.preventDefault();

      const filename = "flagfile.txt";
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
                href="https://github.com/fleetdm/fleet/blob/master/docs/2-Deployment/3-Adding-hosts.md"
                target="_blank"
                rel="noopener noreferrer"
              >
                Add Hosts Documentation <KolideIcon name="external-link" />
              </a>
            </h4>
          </div>
          <ol className={`${baseClass}__install-steps`}>
            <li>
              <h4>
                <span className={`${baseClass}__step-number`}>1</span>Enroll
                secret
              </h4>
              <p>
                Provide an active enroll secret to allow osquery to authenticate
                with the Fleet server:
              </p>
              <div className={`${baseClass}__secret-wrapper`}>
                <EnrollSecretTable secrets={enrollSecret} />
              </div>
            </li>
            <li>
              <h4>
                <span className={`${baseClass}__step-number`}>2</span>Server
                certificate
              </h4>
              <p>
                Provide the TLS certificate used by the Fleet server to enable
                secure connections from osquery:
              </p>
              <p>
                {fetchCertificateError ? (
                  <span className={`${baseClass}__error`}>
                    {fetchCertificateError}
                  </span>
                ) : (
                  <a
                    href="#downloadCertificate"
                    onClick={this.onFetchCertificate}
                  >
                    Download
                    <img src={DownloadIcon} alt="download icon" />
                  </a>
                )}
              </p>
            </li>
            <li>
              <h4>
                <span className={`${baseClass}__step-number`}>3</span>Flagfile
              </h4>
              <p>
                If using the enroll secret and server certificate downloaded
                above, use the generated flagfile. In some configurations,
                modifications may need to be made:
              </p>
              <p>
                <a href="#downloadFlagfile" onClick={onDownloadFlagfile}>
                  Download
                  <img src={DownloadIcon} alt="download icon" />
                </a>
              </p>
            </li>
            <li>
              <h4>
                <span className={`${baseClass}__step-number`}>4</span>Run
                osquery
              </h4>
              <p>
                Run osquery from the directory containing the above files (may
                require sudo or Run as Administrator privileges):
              </p>
              <pre>osqueryd --flagfile=flagfile.txt --verbose</pre>
            </li>
          </ol>
        </div>

        <div className={`${baseClass}__button-wrap`}>
          <Button onClick={onReturnToApp} className="button button--brand">
            Done
          </Button>
        </div>
      </div>
    );
  }
}

export default AddHostModal;
