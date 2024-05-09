/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import {
  createMockGetHostSoftwareResponse,
  createMockHostSoftware,
} from "__mocks__/hostMock";
import { IDeviceUserResponse } from "interfaces/host";
import { IHostSoftware } from "interfaces/software";
import sendRequest from "services";
import endpoints from "utilities/endpoints";

export type ILoadHostDetailsExtension = "device_mapping" | "macadmins";

export interface IGetDeviceSoftwareResponse {
  software: IHostSoftware[];
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
    deviceAuthToken: string
  ): Promise<IGetDeviceSoftwareResponse> => {
    const { DEVICE_SOFTWARE } = endpoints;

    // TODO: remove when API ready
    // return sendRequest("GET", DEVICE_SOFTWARE(deviceAuthToken));

    return new Promise((resolve) => {
      resolve(
        createMockGetHostSoftwareResponse({
          software: [createMockHostSoftware()],
        })
      );
    });
  },
};
