import { IDeviceUserResponse } from "interfaces/host";
import { IDeviceSoftware } from "interfaces/software";
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { buildQueryStringFromParams } from "utilities/url";
import { IHostSoftwareQueryParams } from "./hosts";

export type ILoadHostDetailsExtension = "device_mapping" | "macadmins";

export interface IDeviceSoftwareQueryKey extends IHostSoftwareQueryParams {
  scope: "device_software";
  id: string;
}

export interface IGetDeviceSoftwareResponse {
  software: IDeviceSoftware[];
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
    params: IDeviceSoftwareQueryKey
  ): Promise<IGetDeviceSoftwareResponse> => {
    const { DEVICE_SOFTWARE } = endpoints;
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const { id, scope, ...rest } = params;
    const queryString = buildQueryStringFromParams(rest);
    return sendRequest("GET", `${DEVICE_SOFTWARE(id)}?${queryString}`);
  },

  installSelfServiceSoftware: (
    deviceToken: string,
    softwareTitleId: number
  ) => {
    const { DEVICE_SOFTWARE_INSTALL } = endpoints;
    const path = DEVICE_SOFTWARE_INSTALL(deviceToken, softwareTitleId);

    return sendRequest("POST", path);
  },
};
