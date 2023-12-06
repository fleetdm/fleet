import sendRequest from "services";
import endpoints from "utilities/endpoints";

export interface IGetHQRResponse {
  query_id: number;
  query: string;
  discard_data: boolean;
  host_id: number;
  host_name: string;
  host_team_id: number; // confirm
  last_fetched: string | null; // timestamp
  report_clipped: boolean;
  results: unknown[];
}

export default {
  load: (hostId: number, queryId: number): Promise<IGetHQRResponse> => {
    return sendRequest("GET", endpoints.HOST_QUERY_REPORT(hostId, queryId));
  },
};
