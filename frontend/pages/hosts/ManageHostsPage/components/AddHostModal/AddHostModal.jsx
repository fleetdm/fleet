import React, { Component } from "react";
import PropTypes from "prop-types";
import FileSaver from "file-saver";

import Fleet from "fleet";
import Button from "components/buttons/Button";
import configInterface from "interfaces/config";
import teamInterface from "interfaces/team";
import userInterface from "interfaces/user";
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
    onReturnToApp: PropTypes.func,
    config: configInterface,
    currentUser: userInterface,
  };

  constructor(props) {
    super(props);

    this.userRole = {
      isAnyTeamMaintainer: permissionUtils.isAnyTeamMaintainer(
        this.props.currentUser
      ),
      isGlobalAdmin: permissionUtils.isGlobalAdmin(this.props.currentUser),
      isGlobalMaintainer: permissionUtils.isGlobalMaintainer(
        this.props.currentUser
      ),
    };
    this.currentUserTeams = this.userRole.isAnyTeamMaintainer
      ? Object.values(this.props.currentUser.teams).filter(
          (team) => team.role === "maintainer"
        )
      : this.props.teams;

    this.teamSecrets = Object.values(this.props.teams).map((team) => {
      return { id: team.id, name: team.name, secrets: team.secrets };
    });

    this.state = {
      fetchCertificateError: undefined,
      selectedTeam: null,
      globalSecrets: [],
      selectedEnrollSecrets: [],
    };
  }

  componentDidMount() {
    const { isGlobalAdmin, isGlobalMaintainer } = this.userRole;

    (() => {
      if (isGlobalAdmin || isGlobalMaintainer) {
        Fleet.config
          .loadEnrollSecret()
          .then((response) => {
            this.setState({
              globalSecrets: response.spec.secrets,
              selectedTeam: { id: NO_TEAM_OPTION.value }, // Reset initial selectedTeam value to "no-team" in the case of global users
            });
          })
          .catch((err) => {
            console.log(err);
          });
      } else {
        this.setState({ selectedTeam: this.currentUserTeams[0] });
      }
    })();

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

  // if isGlobalAdmin or isGlobalMaintainer, we include a "No team" option and reveal globalSecrets
  // if not, we pull secrets for the user's teams from the teamsSecrets
  onChangeSelectTeam = (teamId) => {
    const { globalSecrets } = this.state;
    const { currentUserTeams, teamSecrets } = this;
    if (teamId === "no-team") {
      this.setState({ selectedTeam: { id: NO_TEAM_OPTION.value } });
      this.setState({ selectedEnrollSecrets: globalSecrets || [] });
    } else {
      const selectedTeam = currentUserTeams.find((team) => team.id === teamId);
      const selectedEnrollSecrets =
        teamSecrets.find((e) => e.id === selectedTeam.id)?.secrets || "";
      this.setState({ selectedTeam });
      this.setState({
        selectedEnrollSecrets,
      });
    }
  };

  getSelectedEnrollSecrets = (selectedTeam) => {
    if (selectedTeam.id === NO_TEAM_OPTION.value) {
      console.log("this.state.globalSecrets", this.state.globalSecrets);
      return this.state.globalSecrets;
    }
    console.log(
      this.teamSecrets.find((e) => e.id === selectedTeam.id)?.secrets || ""
    );
    return (
      this.teamSecrets.find((e) => e.id === selectedTeam.id)?.secrets || ""
    );
  };

  createTeamDropdownOptions = (currentUserTeams) => {
    const teamOptions = currentUserTeams.map((team) => {
      return {
        value: team.id,
        label: team.name,
      };
    });
    return this.userRole.isAnyTeamMaintainer
      ? teamOptions
      : [NO_TEAM_OPTION, ...teamOptions];
  };

  render() {
    const { config, onReturnToApp } = this.props;
    const { fetchCertificateError, selectedTeam, globalSecrets } = this.state;
    const {
      createTeamDropdownOptions,
      currentUserTeams,
      getSelectedEnrollSecrets,
      onChangeSelectTeam,
    } = this;

    const isBasicTier = permissionUtils.isBasicTier(config);

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
                {isBasicTier ? (
                  <Dropdown
                    wrapperClassName={`${baseClass}__team-dropdown-wrapper`}
                    label={"Select a team for this new host:"}
                    value={selectedTeam && selectedTeam.id}
                    options={createTeamDropdownOptions(currentUserTeams)}
                    onChange={onChangeSelectTeam}
                    placeholder={"Select a team"}
                    searchable={false}
                  />
                ) : null}
                {isBasicTier && selectedTeam && (
                  <EnrollSecretTable
                    secrets={getSelectedEnrollSecrets(selectedTeam)}
                  />
                )}
                {!isBasicTier && <EnrollSecretTable secrets={globalSecrets} />}
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
