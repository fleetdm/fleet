import { IConfig } from "interfaces/config";
import { API_NO_TEAM_ID, ITeamConfig } from "interfaces/team";

export const getManualAgentInstallSetting = (
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
        .macos_enable_bootstrap_package_during_manual_enrollment || false
    );
  }
  return (
    teamConfig?.mdm?.setup_experience
      .macos_enable_bootstrap_package_during_manual_enrollment || false
  );
};

export const getBootstrapPackageMigrationSetting = (
  currentTeamId: number,
  globalConfig?: IConfig,
  teamConfig?: ITeamConfig
) => {
  if (currentTeamId === API_NO_TEAM_ID) {
    return (
      globalConfig?.mdm.setup_experience
        .macos_enable_bootstrap_package_during_migration || false
    );
  }
  return (
    teamConfig?.mdm?.setup_experience
      .macos_enable_bootstrap_package_during_migration || false
  );
};
