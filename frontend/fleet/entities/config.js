import { get } from "lodash";

import endpoints from "fleet/endpoints";
import helpers from "fleet/helpers";

export default (client) => {
  return {
    loadAll: () => {
      const { CONFIG } = endpoints;

      return client.authenticatedGet(client._endpoint(CONFIG));
    },
    loadCertificate: () => {
      const endpoint = client._endpoint("/v1/fleet/config/certificate");

      return client
        .authenticatedGet(endpoint)
        .then((response) => global.window.atob(response.certificate_chain));
    },
    loadEnrollSecret: () => {
      const endpoint = client._endpoint("/v1/fleet/spec/enroll_secret");

      return client.authenticatedGet(endpoint);
    },
    update: (formData) => {
      const { CONFIG } = endpoints;
      const configData = helpers.formatConfigDataForServer(formData);

      if (get(configData, "smtp_settings.port")) {
        configData.smtp_settings.port = parseInt(
          configData.smtp_settings.port,
          10
        );
      }
      return client.authenticatedPatch(
        client._endpoint(CONFIG),
        JSON.stringify(configData)
      );
    },
  };
};
