/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import specAPI from "services/entities/spec";
import teamsAPI from "services/entities/teams";

import {
  IEnrollSecret,
  IEnrollSecretsResponse,
} from "interfaces/enroll_secret";
import { APP_CONTEXT_NO_TEAM_ID } from "interfaces/team";

export default {
  getGlobalEnrollSecrets: () => {
    return specAPI.getEnrollSecretSpec().then((res) => res.spec);
  },
  modifyGlobalEnrollSecrets: (secrets: IEnrollSecret[]) => {
    return specAPI
      .applyEnrollSecretSpec({ spec: { secrets } })
      .then((res) => res.spec);
  },
  getTeamEnrollSecrets: (teamId?: number): Promise<IEnrollSecretsResponse> => {
    if (!teamId || teamId <= APP_CONTEXT_NO_TEAM_ID) {
      return Promise.reject(
        new Error(
          `Invalid team id: ${teamId} must be greater than ${APP_CONTEXT_NO_TEAM_ID}`
        )
      );
    }
    return teamsAPI.getEnrollSecrets(teamId);
  },
  modifyTeamEnrollSecrets: (
    teamId: number | undefined,
    secrets: IEnrollSecret[]
  ): Promise<IEnrollSecretsResponse> => {
    if (!teamId || teamId <= APP_CONTEXT_NO_TEAM_ID) {
      return Promise.reject(
        new Error(
          `Invalid team id: ${teamId} must be greater than ${APP_CONTEXT_NO_TEAM_ID}`
        )
      );
    }
    return teamsAPI.modifyEnrollSecrets(teamId, secrets);
  },
};
