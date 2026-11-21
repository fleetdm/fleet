import sendRequest from "services";
import endpoints from "utilities/endpoints";

export default {
  live_query: () => {
    const { STATUS_LIVE_QUERY } = endpoints;

    return sendRequest("GET", STATUS_LIVE_QUERY);
  },
};
