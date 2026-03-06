import sendRequest from "services";

import endpoints from "utilities/endpoints";

import { API_NO_TEAM_ID, APP_CONTEXT_NO_TEAM_ID } from "interfaces/team";

export interface IUpdateRecoveryLockPasswordResponse {
  enable_recovery_lock_password: boolean;
  fleet_id: number;
}

const recoveryLockPasswordService = {
  updateRecoveryLockPassword: (
    enableRecoveryLockPassword: boolean,
    fleetId?: number
  ): Promise<IUpdateRecoveryLockPasswordResponse> => {
    const { UPDATE_RECOVERY_LOCK_PASSWORD: path } = endpoints;
    return sendRequest("POST", path, {
      enable_recovery_lock_password: enableRecoveryLockPassword,
      fleet_id: fleetId === APP_CONTEXT_NO_TEAM_ID ? API_NO_TEAM_ID : fleetId,
    });
  },
};

export default recoveryLockPasswordService;
