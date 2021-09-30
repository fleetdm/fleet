import sendRequest from "services";
import endpoints from "fleet/endpoints";

export default {
  getCounts: () => {
    const { STATUS_LABEL_COUNTS } = endpoints;

    return sendRequest("GET", STATUS_LABEL_COUNTS);
  },
};
