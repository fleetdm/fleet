/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "fleet/endpoints";
// import { IConfig } from "interfaces/host";

// TODO add other methods from "fleet/entities/config"

export default {
  loadAll: () => {
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
};
