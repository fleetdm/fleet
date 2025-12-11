/*
 * NOTE: Do not make changes to this file!
 * Also please check the README for how to use the mock service :)
 */

import { trim } from "lodash";

import CONFIG, { MockEndpointHandler } from "../mocks/config";

const {
  DELAY,
  ENDPOINT,
  REQUEST_RESPONSE_MAPPINGS: RESPONSES,
  WILDCARDS,
} = CONFIG;

export const sleep = (ms: number) =>
  new Promise((resolve) => setTimeout(resolve, ms));

const splitRouteAndQueryString = (path: string) => {
  path = trim(path, "/").replace(trim(ENDPOINT, "/"), "");
  const strings = trim(path, "/").split("?");

  if (strings.length > 2) {
    throw new Error(
      "Invalid usage: URL cannot contain more than one `?` and query string must follow format `?key=value&another_key=another_value`"
    );
  }
  if (strings.length === 1) {
    return path.includes("?")
      ? [undefined, strings[0]]
      : [strings[0], undefined];
  }

  return strings;
};

const getParts = (pathString: string) => {
  const [routeString, queryString] = splitRouteAndQueryString(pathString);
  const routeParts = routeString ? routeString.split("/") : [];
  const queryParts = queryString ? queryString.split("&") : [];

  return routeParts.concat(queryParts);
};

const partsByPathByMethod = {} as Record<string, Record<string, string[]>>;
Object.entries(RESPONSES).forEach(([method, paths]) => {
  Object.keys(paths).forEach((pathString) => {
    if (!partsByPathByMethod[method]) {
      partsByPathByMethod[method] = {} as Record<string, string[]>;
    }
    partsByPathByMethod[method][pathString] = getParts(pathString);
  });
});

const isPartMatch = (
  partToMatch: string,
  configPart: string,
  wildcards: string[] = []
) => {
  return (
    partToMatch === configPart || wildcards.some((w) => configPart.includes(w)) // if a config part includes any wildcards, it matches with any value
  );
};

const matchPathToResponse = (method: string, requestPath: string) => {
  const results = Object.entries(partsByPathByMethod[method]).filter(
    ([, configParts]) => {
      const requestParts = getParts(requestPath);
      return (
        requestParts.length === configParts.length &&
        requestParts.every((p, i) => isPartMatch(p, configParts[i], WILDCARDS))
      );
    }
  );

  return results;
};

export const sendRequest = async (
  method = "GET",
  requestPath: string,
  data?: unknown
): Promise<any> => {
  console.log("Mock service request URL: ", requestPath);
  console.log("Mock service request body: ", data);

  requestPath = trim(requestPath, "/").replace(ENDPOINT, "");
  let response:
    | Record<string, unknown>
    | ((requestPath: string, data?: unknown) => Record<string, unknown>)
    | undefined;
  let responseKey: string | undefined;

  try {
    const matches = matchPathToResponse(method, requestPath) || [];
    if (matches.length > 1) {
      [responseKey] =
        matches.find(([key]) => !WILDCARDS.some((w) => key.includes(w))) || [];
    } else {
      responseKey = matches?.[0]?.[0];
    }
  } catch (err) {
    return Promise.reject(err);
  }

  if (responseKey) {
    const methodHandlers = RESPONSES[method] as
      | Record<string, MockEndpointHandler>
      | undefined;
    const handler = methodHandlers?.[responseKey];

    if (typeof handler === "function") {
      response = await handler(requestPath, data);
    } else {
      response = handler;
    }
  }

  if (!responseKey || !response) {
    return Promise.reject(`Mock service error: 404 ${requestPath} not found`);
  }

  await sleep(DELAY);
  console.log("Mock service response: ", response);

  return Promise.resolve(response);
};

export default sendRequest;
