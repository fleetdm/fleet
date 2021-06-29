import React, { Component } from "react";
import PropTypes from "prop-types";
import FileSaver from "file-saver";

import Fleet from "fleet";
import Button from "components/buttons/Button";
import configInterface from "interfaces/config";
import teamInterface from "interfaces/team";
import enrollSecretInterface from "interfaces/enroll_secret";
import permissionUtils from "utilities/permissions";
import EnrollSecretTable from "components/config/EnrollSecretTable";
import FleetIcon from "components/icons/FleetIcon";
import Dropdown from "components/forms/fields/Dropdown";
import DownloadIcon from "../../../../../../assets/images/icon-download-12x12@2x.png";

const baseClass = "add-host-modal";

const NO_TEAM_OPTION = {
  value: "no-team",
  label: "No team",
};

class AddHostModal extends Component {
  static propTypes = {
    teams: PropTypes.arrayOf(teamInterface),
    onChangeTeam: PropTypes.func,
    onReturnToApp: PropTypes.func,
    enrollSecret: enrollSecretInterface,
    config: configInterface,
  };

  constructor(props) {
    super(props);
    this.state = { fetchCertificateError: undefined, selectedTeam: null };
  }

  componentDidMount() {
    Fleet.config
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

  onChangeSelectTeam = (teamId) => {
    const { teams, onChangeTeam } = this.props;
    if (teamId === "no-team") {
      onChangeTeam(null);
      this.setState({ selectedTeam: { id: NO_TEAM_OPTION.value } });
    } else {
      const selectedTeam = teams.find((team) => team.id === teamId);
      onChangeTeam(selectedTeam);
      this.setState({ selectedTeam });
    }
  };

  createTeamDropdownOptions = (teams) => {
    const teamOptions = teams.map((team) => {
      return {
        value: team.id,
        label: team.name,
      };
    });
    return [NO_TEAM_OPTION, ...teamOptions];
  };

  render() {
    const { config, onReturnToApp, enrollSecret, teams } = this.props;
    const { fetchCertificateError, selectedTeam } = this.state;
    const { createTeamDropdownOptions, onChangeSelectTeam } = this;

    let tlsHostname = config.server_url;
    try {
      const serverUrl = new URL(config.server_url);
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
                href="https://github.com/fleetdm/fleet/blob/2f42c281f98e39a72ab4a5125ecd26d303a16a6b/docs/1-Using-Fleet/4-Adding-hosts.md"
                target="_blank"
                rel="noopener noreferrer"
              >
                Add Hosts Documentation <FleetIcon name="external-link" />
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
                Osquery uses an enroll secret to authenticate with the Fleet
                server.
              </p>
              <div className={`${baseClass}__secret-wrapper`}>
                {permissionUtils.isBasicTier(config) ? (
                  <Dropdown
                    wrapperClassName={`${baseClass}__team-dropdown-wrapper`}
                    label={"Select a team for this new host:"}
                    value={selectedTeam && selectedTeam.id}
                    options={createTeamDropdownOptions(teams)}
                    onChange={onChangeSelectTeam}
                    placeholder={"Select a team"}
                    searchable={false}
                  />
                ) : null}
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
