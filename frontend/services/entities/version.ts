 
import sendRequest from "services";
import endpoints from "utilities/endpoints";

export default {
  load: () => {
    const { VERSION } = endpoints;

    return sendRequest("GET", VERSION);
  },
};
