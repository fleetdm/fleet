 
import sendRequest from "services";
import endpoints from "utilities/endpoints";

export default {
  getCounts: () => {
    const { STATUS_LABEL_COUNTS } = endpoints;

    return sendRequest("GET", STATUS_LABEL_COUNTS);
  },
};
