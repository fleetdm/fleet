/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "fleet/endpoints";

export default {
  getSummary: (teamId: number | undefined) => {
    const { HOST_SUMMARY } = endpoints;

    return sendRequest("GET", HOST_SUMMARY(teamId));
  },
};
