import sendRequest from "services";

import endpoints from "utilities/endpoints";

import { API_NO_TEAM_ID, APP_CONTEXT_NO_TEAM_ID } from "interfaces/team";

const recoveryLockPasswordService = {
  updateRecoveryLockPassword: (
    enableRecoveryLockPassword: boolean,
    fleetId?: number
  ) => {
    const { UPDATE_RECOVERY_LOCK_PASSWORD: path } = endpoints;
    return sendRequest("POST", path, {
      enable_recovery_lock_password: enableRecoveryLockPassword,
      fleet_id: fleetId === APP_CONTEXT_NO_TEAM_ID ? API_NO_TEAM_ID : fleetId,
    });
  },
};

export default recoveryLockPasswordService;
