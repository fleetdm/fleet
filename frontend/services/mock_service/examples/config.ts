/*
 * NOTE: This is an example of how to configure your mock service.
 * Be sure to copy this file into `../mocks` and only edit that copy!
 * Also please check the README for how to use the mock service :)
 */

import RESPONSES from "./responses";

type IResponses = Record<string, Record<string, Record<string, unknown>>>;

const DELAY = 1000; // modify the DELAY value (in milliseconds) to simulate a delayed async response

const ENDPOINT = "/latest/fleet"; // modify the ENDPOINT string to correspond to your API spec

// WILDCARDS can be used to represent URL parameters in any combination as illustrated below
// modify the WILDCARDS array if you prefer to use different characters
const WILDCARDS: string[] = [":", "*", "{", "}"];

// REQUEST_RESPONSE_MAPPINGS dictionary maps your static responses to the specified API request path
const REQUEST_RESPONSE_MAPPINGS: IResponses = {
  GET: {
    // this is a basic path with no wildcards
    "/hosts?page=0&per_page=20&order_key=display_name&order_direction=asc":
      RESPONSES.ALL_HOSTS,
    // this basic path only matches with '1337' as the value for the team id query param
    "/hosts?page=0&per_page=20&order_key=display_name&order_direction=asc&team_id=1337":
      RESPONSES.HOSTS_TEAM_1337,
    // this wildcard path matches with any other value for the team id query param
    "/hosts?page=0&per_page=20&order_key=display_name&order_direction=asc&team_id={team_id}":
      RESPONSES.HOSTS_TEAM_ID,
    // this basic path only matches with '1337' as the value for the host id route param
    "/hosts/1337": RESPONSES.HOST_1337,
    // this wildcard path matches with any other value for the host id route param
    "/hosts/*id": RESPONSES.HOST_ID,
    // this wildcard path matches with any value for the host id route param
    "/hosts/:id/device_mapping": RESPONSES.DEVICE_MAPPING,
    // this wildcard path matches with any value for the host id route param
    "hosts/{*}/macadmins": RESPONSES.MACADMINS,
    // this is a basic path with no wildcards
    "hosts/count": {
      count: 1,
    },
    // this wildcard path matches with any value for the team id route param
    "hosts/count?team_id={*}": {
      count: 1,
    },
  },
  // additional mappings can be specified for other HTTP request types (POST, PATCH, DELETE, etc.)
  POST: {
    "/:id/refetch": {}, // this wildcard route returns empty JSON
  },
} as IResponses;

export default { DELAY, ENDPOINT, WILDCARDS, REQUEST_RESPONSE_MAPPINGS };
