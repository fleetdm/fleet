import { IConfig } from "interfaces/config";
import { API_NO_TEAM_ID, ITeamConfig } from "interfaces/team";

const getManualAgentInstallSetting = (
  currentTeamId: number,
  globalConfig?: IConfig,
  teamConfig?: ITeamConfig
) => {
  if (currentTeamId === API_NO_TEAM_ID) {
    return (
      globalConfig?.mdm.setup_experience.macos_manual_agent_install || false
    );
  }
  return teamConfig?.mdm?.setup_experience.macos_manual_agent_install || false;
};

export const getBootstrapPackageManualEnrollmentSetting = (
  currentTeamId: number,
  globalConfig?: IConfig,
  teamConfig?: ITeamConfig
) => {
  if (currentTeamId === API_NO_TEAM_ID) {
    return (
      globalConfig?.mdm.setup_experience
        .macos_bootstrap_package_manual_enrollment || false
    );
  }
  return (
    teamConfig?.mdm?.setup_experience
      .macos_bootstrap_package_manual_enrollment || false
  );
};

export default getManualAgentInstallSetting;
