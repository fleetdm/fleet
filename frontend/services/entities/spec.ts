/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "fleet/endpoints";
import { IEnrollSecret } from "interfaces/enroll_secret";

interface IEnrollSecretSpec {
  spec: {
    secrets: IEnrollSecret[];
  };
}

export default {
  getEnrollSecretSpec: () => {
    const { GLOBAL_ENROLL_SECRETS } = endpoints;

    return sendRequest("GET", GLOBAL_ENROLL_SECRETS);
  },
  applyEnrollSecretSpec: (spec: IEnrollSecretSpec) => {
    const { GLOBAL_ENROLL_SECRETS } = endpoints;

    return sendRequest("POST", GLOBAL_ENROLL_SECRETS, spec);
  },
};
