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
import Dropdown from "components/forms/fields/Dropdown";

const baseClass = "enroll-secret-modal";

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
      return this.state.globalSecrets;
    }
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
    const { selectedTeam, globalSecrets } = this.state;
    const {
      createTeamDropdownOptions,
      currentUserTeams,
      getSelectedEnrollSecrets,
      onChangeSelectTeam,
    } = this;

    const isBasicTier = permissionUtils.isBasicTier(config);

    // let tlsHostname = config.server_url;
    // try {
    //   const serverUrl = new URL(config.server_url);
    //   tlsHostname = serverUrl.hostname;
    //   if (serverUrl.port) {
    //     tlsHostname += `:${serverUrl.port}`;
    //   }
    // } catch (e) {
    //   if (!(e instanceof TypeError)) {
    //     throw e;
    //   }
    // }

    return (
      <div className={baseClass}>
        <div className={`${baseClass}__description`}>
          Use these secret(s) to enroll devices to WHAT TEAM THEY'RE VIEWING
        </div>
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
