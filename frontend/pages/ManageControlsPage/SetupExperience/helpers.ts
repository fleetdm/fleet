import { IConfig } from "interfaces/config";
import { API_NO_TEAM_ID, ITeamConfig } from "interfaces/team";

const getManualAgentInstallSetting = (
  currentTeamId: number,
  globalConfig?: IConfig,
  teamConfig?: ITeamConfig
) => {
  if (currentTeamId === API_NO_TEAM_ID) {
    return globalConfig?.mdm.macos_setup.manual_agent_install || false;
  }
  return teamConfig?.mdm?.macos_setup.manual_agent_install || false;
};

export default getManualAgentInstallSetting;
