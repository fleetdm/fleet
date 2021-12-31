/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import { get } from "lodash";

import sendRequest from "services";
import endpoints from "fleet/endpoints";
import helpers from "fleet/helpers";
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
  update: (formData: any) => {
    const { CONFIG } = endpoints;

    // Failing policies webhook does not use flatten <> nest config helper
    if (formData.webhook_settings.failing_policies_webhook) {
      return sendRequest("PATCH", CONFIG, formData);
    }

    const configData = helpers.formatConfigDataForServer(formData);

    if (get(configData, "smtp_settings.port")) {
      configData.smtp_settings.port = parseInt(
        configData.smtp_settings.port,
        10
      );
    }
    return sendRequest("PATCH", CONFIG, configData);
  },
};
