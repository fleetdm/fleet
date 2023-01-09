/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import { sendRequest } from "services/mock_service/service/service"; // MDM TODO: Replace when backend is merged
// import sendRequest from "services";
import endpoints from "utilities/endpoints";

export default {
  downloadEnrollmentProfile: () => {
    const { MDM_DOWNLOAD_ENROLLMENT_PROFILE } = endpoints;
    return sendRequest("GET", MDM_DOWNLOAD_ENROLLMENT_PROFILE);
  },
};
