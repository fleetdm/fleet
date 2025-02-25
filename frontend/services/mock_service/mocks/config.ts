/*
 * NOTE: This is an example of how to configure your mock service.
 * Be sure to copy this file into `../mocks` and only edit that copy!
 * Also please check the README for how to use the mock service :)
 */

import RESPONSES from "./responses";

type IResponses = Record<string, Record<string, Record<string, unknown>>>;

const DELAY = 5000;

const ENDPOINT = "/latest/fleet";

const WILDCARDS: string[] = [":", "*", "{", "}"];

const REQUEST_RESPONSE_MAPPINGS: IResponses = {
  GET: {
    // response is list of all labels excluding any expensive data operations (UI only needs label
    // name and id for this page)
    "labels?summary=true": RESPONSES.labels,
    // request query string is hostname, uuid, or mac address; response is host detail excluding any
    // expensive data operations
    "targets?query={*}": RESPONSES.hosts,
    // "SchedulableQueries" to be used in developing frontend for #7765
    "hosts/12345": RESPONSES.hostDetailsiOS,
    queries: RESPONSES.globalQueries,
    "queries/1": RESPONSES.globalQuery1,
    "queries/2": RESPONSES.globalQuery2,
    "queries/3": RESPONSES.globalQuery3,
    "queries/4": RESPONSES.teamQuery1,
    "queries/5": RESPONSES.globalQuery4,
    "queries/6": RESPONSES.globalQuery5,
    "queries/7": RESPONSES.globalQuery6,
    "queries/8": RESPONSES.teamQuery2,
    "queries?team_id=13": RESPONSES.teamQueries,
    "queries/113/report?order_key=host_name&order_direction=asc":
      RESPONSES.queryReport,
  },
  POST: {
    // request body is ISelectedTargets
    "targets/count": {
      targets_count: 1,
      targets_online: 0,
      targets_offline: 1,
      targets_missing_in_action: 0,
    },
    // "SchedulableQueries" to be used in developing frontend for #7765
    queries: {
      description: "Ok",
      name: "New query name",
      observer_can_run: false,
      query: "SELECT * FROM osquery_info;",
      id: 1,
      team_id: null,
      platform: "linux",
    },
    "autofill/policies": RESPONSES.aiAutofillPolicy,
  },
} as IResponses;

export default { DELAY, ENDPOINT, WILDCARDS, REQUEST_RESPONSE_MAPPINGS };
