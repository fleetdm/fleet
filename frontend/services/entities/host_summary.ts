/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { buildQueryStringFromParams } from "utilities/url";

interface ISummaryProps {
  teamId?: number;
  platform?: string;
  lowDiskSpace?: number;
}

export default {
  getSummary: ({ teamId, platform, lowDiskSpace }: ISummaryProps) => {
    const queryParams = {
      team_id: teamId,
      platform,
      low_disk_space: lowDiskSpace,
    };

    const queryString = buildQueryStringFromParams(queryParams);
    const endpoint = endpoints.HOST_SUMMARY;
    const path = `${endpoint}?${queryString}`;

    return sendRequest("GET", path);
  },
};
