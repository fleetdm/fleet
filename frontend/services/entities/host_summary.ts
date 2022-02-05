/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "fleet/endpoints";

interface ISummaryProps {
  teamId?: number;
  platform?: string;
}

export default {
  getSummary: ({
    teamId,
    platform,
  }: ISummaryProps) => {
    const { HOST_SUMMARY } = endpoints;
    let queryString = "";

    if (teamId) {
      queryString += `&team_id=${teamId}`;
    }

    // platform can be empty string
    if (!!platform) {
      queryString += `&platform=${platform}`;
    }

    // Append query string to endpoint route after slicing off the leading ampersand
    const path = `${HOST_SUMMARY}${queryString && `?${queryString.slice(1)}`}`;
    return sendRequest("GET", path);
  },
};
