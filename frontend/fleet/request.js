import fetch from "isomorphic-fetch";
import { isUndefined, omitBy } from "lodash";

export default class Request {
  constructor(options) {
    this.body = options.body;
    this.credentials = "same-origin";
    this.endpoint = options.endpoint;
    this.headers = {
      Accept: "application/json",
      "Content-Type": "application/json",
      ...options.headers,
    };
    this.method = options.method;
  }

  static REQUEST_METHODS = {
    DELETE: "DELETE",
    GET: "GET",
    PATCH: "PATCH",
    POST: "POST",
  };

  static handleResponse(response, jsonResponse) {
    if (response.ok) {
      return jsonResponse;
    }

    const error = new Error(response.statusText);
    error.error = jsonResponse.error;
    error.message = jsonResponse;
    error.response = jsonResponse;
    error.status = response.status;

    throw error;
  }

  get requestAttributes() {
    const { body, credentials, headers, method } = this;

    return omitBy({ body, credentials, headers, method }, isUndefined);
  }

  send() {
    const { endpoint, requestAttributes } = this;

    return fetch(endpoint, requestAttributes).then((response) => {
      return response.json().then((jsonResponse) => {
        return Request.handleResponse(response, jsonResponse);
      });
    });
  }
}
