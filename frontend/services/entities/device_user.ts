import { IDeviceUserResponse } from "interfaces/host";
import { IHostSoftware } from "interfaces/software";
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { buildQueryStringFromParams } from "utilities/url";

export type ILoadHostDetailsExtension = "device_mapping" | "macadmins";

export type IDeviceSoftwareQueryParams = {
  page: number;
  per_page: number;
  query: string;
  order_key: string;
  order_direction: "asc" | "desc";
};

export interface IGetDeviceSoftwareResponse {
  software: IHostSoftware[];
  count: number;
  meta: {
    has_next_results: boolean;
    has_previous_results: boolean;
  };
}

export default {
  loadHostDetails: (deviceAuthToken: string): Promise<IDeviceUserResponse> => {
    const { DEVICE_USER_DETAILS } = endpoints;
    const path = `${DEVICE_USER_DETAILS}/${deviceAuthToken}`;

    return sendRequest("GET", path);
  },
  loadHostDetailsExtension: (
    deviceAuthToken: string,
    extension: ILoadHostDetailsExtension
  ) => {
    const { DEVICE_USER_DETAILS } = endpoints;
    const path = `${DEVICE_USER_DETAILS}/${deviceAuthToken}/${extension}`;

    return sendRequest("GET", path);
  },
  refetch: (deviceAuthToken: string) => {
    const { DEVICE_USER_DETAILS } = endpoints;
    const path = `${DEVICE_USER_DETAILS}/${deviceAuthToken}/refetch`;

    return sendRequest("POST", path);
  },

  getDeviceSoftware: (
    deviceAuthToken: string,
    params: IDeviceSoftwareQueryParams
  ): Promise<IGetDeviceSoftwareResponse> => {
    const { DEVICE_SOFTWARE } = endpoints;
    const queryString = buildQueryStringFromParams(params as any); // TODO: fix with generics
    return sendRequest(
      "GET",
      `${DEVICE_SOFTWARE(deviceAuthToken)}?${queryString}`
    );
  },
};
