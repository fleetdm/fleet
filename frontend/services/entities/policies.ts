/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */

import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { IStoredPolicyResponse } from "interfaces/policy";

export default {
  load: (id: number): Promise<IStoredPolicyResponse> => {
    const { GLOBAL_POLICIES } = endpoints;
    const path = `${GLOBAL_POLICIES}/${id}`;

    return sendRequest("GET", path);
  },
};
