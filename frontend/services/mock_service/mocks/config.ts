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
    config: RESPONSES.config1, // just first integration -- to throw error, rename config as configz
  },
  // additional mappings can be specified for other HTTP request types (POST, PATCH, DELETE, etc.)
  PATCH: {
    config: RESPONSES.configAdd2, // will add second integration to first one
  },
  DELETE: {
    // will remove second integration
    config: RESPONSES.config1,
  },
} as IResponses;

export default { DELAY, ENDPOINT, WILDCARDS, REQUEST_RESPONSE_MAPPINGS };
