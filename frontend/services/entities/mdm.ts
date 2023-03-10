/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { buildQueryStringFromParams } from "utilities/url";

export default {
  downloadDeviceUserEnrollmentProfile: (token: string) => {
    const { DEVICE_USER_MDM_ENROLLMENT_PROFILE } = endpoints;
    return sendRequest("GET", DEVICE_USER_MDM_ENROLLMENT_PROFILE(token));
  },
  unenrollHostFromMdm: (hostId: number, timeout?: number) => {
    const { HOST_MDM_UNENROLL } = endpoints;
    return sendRequest(
      "PATCH",
      HOST_MDM_UNENROLL(hostId),
      undefined,
      undefined,
      timeout
    );
  },
  requestCSR: (email: string, organization: string) => {
    const { MDM_REQUEST_CSR } = endpoints;

    return sendRequest("POST", MDM_REQUEST_CSR, {
      email_address: email,
      organization,
    });
  },

  getProfiles: (teamId?: number) => {
    const { MDM_PROFILES } = endpoints;

    let path = MDM_PROFILES;

    if (teamId) {
      path = `${path}?${buildQueryStringFromParams({ team_id: teamId })}`;
    }

    return sendRequest("GET", path);
  },

  uploadProfile: (file: File, teamId?: number) => {
    const { MDM_PROFILES } = endpoints;

    const formData = new FormData();
    formData.append("profile", file);

    if (teamId) {
      formData.append("team_id", teamId.toString());
    }

    return sendRequest("POST", MDM_PROFILES, formData);
  },

  downloadProfile: (profileId: number) => {
    const { MDM_PROFILE } = endpoints;
    return sendRequest("GET", MDM_PROFILE(profileId));
  },

  deleteProfile: (profileId: number) => {
    const { MDM_PROFILE } = endpoints;
    return sendRequest("DELETE", MDM_PROFILE(profileId));
  },

  getAggregateProfileStatuses: (teamId?: number) => {
    let { MDM_PROFILES_AGGREGATE_STATUSES: path } = endpoints;

    if (teamId) {
      path = `${path}?${buildQueryStringFromParams({ team_id: teamId })}`;
    }

    return sendRequest("GET", path);
  },
};
