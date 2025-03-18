import PropTypes from "prop-types";
import { IHost } from "./host";

export default PropTypes.shape({
  uiHostCounts: PropTypes.shape({
    total: PropTypes.number,
    successful: PropTypes.number, // Does not include ChromeOS results that are partially successful
    failed: PropTypes.number,
  }),
  id: PropTypes.number,
  online: PropTypes.number,
});

export interface ICampaignError {
  host_display_name: string;
  osquery_version: string;
  error: string;
}

export interface IUIHostCounts {
  total: number; // number of hosts that responded at all, either with results, no results, or an error
  successful: number; // Number of hosts that responded with results or no results. Excludes hosts that responded with an error. Does not include ChromeOS results that are partially successful
  failed: number; // number of hosts that responded with an error - equivalent to `campaign.errors.length`
}

export interface IServerHostCounts {
  countOfHostsWithResults: number; // Number of hosts that responded with any results
  countOfHostsWithNoResults: number; // Number of hosts that have responded with no results, excluding errors
}

export interface IHostWithQueryResults extends IHost {
  query_results: QueryResults;
}

type QueryResults = Record<string, unknown>[];

export interface ICampaign {
  // upstream websocket and services methods return any
  // so narrower typing at this level is not actually guaranteed
  Metrics?: {
    [key: string]: any;
  };
  created_at: string;

  // totals is a summary of data the server knows about hosts targeted by this campaign, reported at
  // the outset of the live query campaign via a "totals"-type websocket message
  totals: {
    count: number;
    missing_in_action: number;
    offline: number;
    online: number;
  };

  // errors, hosts, hosts_count, and query_results are constructed and updated by the UI from query campaign data as it streams
  // in via "result"-type websocket messages. There is significant overlap of the data contained within
  // these fields.
  errors: ICampaignError[];
  hosts: IHostWithQueryResults[]; // array of all hosts that responded and their associated results
  uiHostCounts: IUIHostCounts; // aggregate data about the results of the live query campaign. Differs from server_host_counts in that this object is constructed and updated by the UI
  queryResults: QueryResults;

  // status and server_host_counts represent information about the state of a live query campaign
  // and the hosts it targets, reported via "status"-type websocket messages.
  status: string; // really "" | "pending" | "finished";
  serverHostCounts: IServerHostCounts; // differs from hosts_count in that this object is tracked and reported by the server

  id: number;
  query_id: number;
  updated_at: string;
  user_id: number;
}

// TODO: review use of ICampaignState to see if legacy code can be removed
export interface ICampaignState {
  campaign: ICampaign;
  observerShowSql: boolean;
  queryIsRunning: boolean;
  queryPosition: {
    [key: string]: any;
  };
  queryResultsToggle: any;
  runQueryMilliseconds: number;
  selectRelatedHostTarget: boolean;
  targetsCount: number;
  targetsError: any;
}
