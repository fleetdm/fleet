/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import specAPI from "services/entities/spec";
import teamsAPI from "services/entities/teams";

import { IEnrollSecret } from "interfaces/enroll_secret";

interface IEnrollSecretSpec {
  spec: {
    secrets: IEnrollSecret[];
  };
}

export default {
  getGlobalEnrollSecrets: () => {
    return specAPI.getEnrollSecretSpec().then((res) => res.spec);
  },
  modifyGlobalEnrollSecrets: (secrets: IEnrollSecret[]) => {
    return specAPI
      .applyEnrollSecretSpec({ spec: { secrets } })
      .then((res) => res.spec);
  },
  getTeamEnrollSecrets: (teamId: number) => {
    return teamsAPI.getEnrollSecrets(teamId);
  },
  modifyTeamEnrollSecrets: (teamId: number, secrets: IEnrollSecret[]) => {
    return teamsAPI.modifyEnrollSecrets(teamId, secrets);
  },
};
