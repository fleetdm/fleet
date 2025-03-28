import sendRequest from "services";
import endpoints from "utilities/endpoints";

export interface IGetSCIMDetailsResponse {
  last_request: {
    requested_at: string;
    status: string;
    details: string;
  } | null;
}

export default {
  getSCIMDetails: (): Promise<IGetSCIMDetailsResponse> => {
    const { SCIM_DETAILS } = endpoints;
    return sendRequest("GET", SCIM_DETAILS);
  },
};
