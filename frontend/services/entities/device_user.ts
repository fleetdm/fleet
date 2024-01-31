/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import { IDeviceUserResponse } from "interfaces/host";
import sendRequest from "services";
import endpoints from "utilities/endpoints";

export type ILoadHostDetailsExtension = "device_mapping" | "macadmins";

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
};
