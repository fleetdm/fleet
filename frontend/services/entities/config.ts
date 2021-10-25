/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "fleet/endpoints";
// import { IConfig } from "interfaces/host";
import { IEnrollSecret } from "interfaces/enroll_secret";

// TODO add other methods from "fleet/entities/config"

export default {
  loadAll: () => {
    const { CONFIG } = endpoints;
    const path = `${CONFIG}`;

    return sendRequest("GET", path);
  },
  loadEnrollSecret: () => {
    const { ENROLL_SECRET } = endpoints;
    const path = `${ENROLL_SECRET}`;

    return sendRequest("GET", path);
  },
  newEnrollSecret: (newEnrollSecrets: any) => {
    const { CONFIG } = endpoints;
    const path = `${CONFIG}`;

    return sendRequest("PATCH", path, newEnrollSecrets);
  },
  removeEnrollSecret: (
    globalSecret: IEnrollSecret[],
    selectedSecret: IEnrollSecret
  ) => {
    const { CONFIG } = endpoints;
    const path = `${CONFIG}`;

    // TODO: Return enroll secrets that do not have the same secret as the removed one
    const updatedEnrollSecrets = globalSecret.filter((secret) => {
      return selectedSecret.secret !== secret.secret;
    });

    // TODO: Check if this is how the API works, post enroll secrets
    return sendRequest("POST", path, updatedEnrollSecrets);
  },
};
