/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "fleet/endpoints";

export type ILoadHostDetailsExtension = "device_mapping" | "macadmins";

export default {
  loadHostDetails: (deviceAuthToken: any) => {
    const { DEVICE_USER_DETAILS } = endpoints;
    const path = `${DEVICE_USER_DETAILS}/${deviceAuthToken}`;

    return sendRequest("GET", path);
  },
  loadHostDetailsExtension: (
    deviceAuthToken: any,
    extension: ILoadHostDetailsExtension
  ) => {
    const { DEVICE_USER_DETAILS } = endpoints;
    const path = `${DEVICE_USER_DETAILS}/${deviceAuthToken}/${extension}`;

    return sendRequest("GET", path);
  },
  refetch: (deviceAuthToken: any) => {
    const { DEVICE_USER_DETAILS } = endpoints;
    const path = `${DEVICE_USER_DETAILS}/${deviceAuthToken}/refetch`;

    return sendRequest("POST", path);
  },
};
