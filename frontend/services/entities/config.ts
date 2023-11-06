/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */

import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { IConfig, IMdmConfig } from "interfaces/config";
import axios, { AxiosError } from "axios";

export default {
  loadAll: (): Promise<IConfig> => {
    const { CONFIG } = endpoints;
    const path = `${CONFIG}`;

    return sendRequest("GET", path);
  },
  loadCertificate: () => {
    const { CONFIG } = endpoints;
    const path = `${CONFIG}/certificate`;

    return sendRequest("GET", path).then(({ certificate_chain }) => {
      let decodedCertificate;
      try {
        decodedCertificate = global.window.atob(certificate_chain);
      } catch (err) {
        return Promise.reject(`Unable to decode certificate: ${err}`);
      }
      if (!decodedCertificate) {
        return Promise.reject("Missing or undefined certificate.");
      }

      return Promise.resolve(decodedCertificate);
    });
  },
  loadEnrollSecret: () => {
    const { GLOBAL_ENROLL_SECRETS } = endpoints;

    return sendRequest("GET", GLOBAL_ENROLL_SECRETS);
  },
  update: (formData: any) => {
    const { CONFIG } = endpoints;

    return sendRequest("PATCH", CONFIG, formData);
  },

  /**
   * updateMDMConfig is a special case of update that is used to update the MDM
   * config. It differs in that it allows the caller to parse error responses.
   * If the request fails, the full AxiosError object will be passed as the
   * rejection value instead of a simple string.
   */
  updateMDMConfig: (mdm: Partial<IMdmConfig>) => {
    const { CONFIG } = endpoints;

    return sendRequest("PATCH", CONFIG, { mdm }, undefined, undefined, true);
  },

  // This API call is made to a specific endpoint that is different than our
  // other ones. This is why we have implmented the call with axios here instead
  // of using our sendRequest method.
  loadSandboxExpiry: async () => {
    const instanceId = window.location.host.split(".")[0];
    const url = "https://sandbox.fleetdm.com/expires";

    try {
      const { data } = await axios.get<{ timestamp: string }>(url, {
        url,
        params: { id: instanceId },
        responseType: "json",
      });
      return data.timestamp;
    } catch (error) {
      const axiosError = error as AxiosError;
      return axiosError.response;
    }
  },
};
