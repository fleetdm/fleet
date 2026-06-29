import sendRequest from "services";
import endpoints from "utilities/endpoints";

import { IApiEndpoint } from "interfaces/api_endpoint";

export interface IListApiEndpointsResponse {
  api_endpoints: IApiEndpoint[];
}

export default {
  loadAll: async (): Promise<IApiEndpoint[]> => {
    const { REST_API_ENDPOINTS } = endpoints;

    const response: IListApiEndpointsResponse = await sendRequest(
      "GET",
      REST_API_ENDPOINTS
    );
    return response.api_endpoints;
  },
};
