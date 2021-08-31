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
};
