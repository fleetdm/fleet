/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";

export default {
  downloadDeviceUserEnrollmentProfile: (token: string) => {
    const { DEVICE_USER_MDM_ENROLLMENT_PROFILE } = endpoints;
    return sendRequest("GET", DEVICE_USER_MDM_ENROLLMENT_PROFILE(token));
  },
  unenrollHostFromMdm: (hostId: number) => {
    const { HOST_MDM_UNENROLL } = endpoints;
    return sendRequest("PATCH", HOST_MDM_UNENROLL(hostId));
  },
};
