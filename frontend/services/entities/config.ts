/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */

import sendRequest from "services";
import endpoints from "fleet/endpoints";
import { IConfig } from "interfaces/config";

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
};
