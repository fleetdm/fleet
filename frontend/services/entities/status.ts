import sendRequest from "services";
import endpoints from "utilities/endpoints";

export default {
  result_store: () => {
    const { STATUS_RESULT_STORE } = endpoints;

    return sendRequest("GET", STATUS_RESULT_STORE);
  },
  live_query: () => {
    const { STATUS_LIVE_QUERY } = endpoints;

    return sendRequest("GET", STATUS_LIVE_QUERY);
  },
};
