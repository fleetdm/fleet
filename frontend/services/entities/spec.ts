/* eslint-disable @typescript-eslint/explicit-module-boundary-types */

import { IEnrollSecret } from "interfaces/enroll_secret";
import sendRequest from "services";
import endpoints from "utilities/endpoints";

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
