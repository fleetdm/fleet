import local from "utilities/local";
import Request from "fleet/request";
import URL_PREFIX from "router/url_prefix";

class Base {
  constructor() {
    const { origin } = global.window.location;

    this.baseURL = `${origin}${URL_PREFIX}/api`;
    this.bearerToken = local.getItem("auth_token");
  }

  static post(endpoint, body = {}, headers = {}) {
    const { POST } = Request.REQUEST_METHODS;

    return Base._request(POST, endpoint, body, headers);
  }

  static get(endpoint, overrideHeaders = {}) {
    const { GET } = Request.REQUEST_METHODS;
    return Base._request(GET, endpoint, {}, overrideHeaders);
  }

  static _deleteRequest(endpoint, headers, body) {
    const { DELETE: method } = Request.REQUEST_METHODS;
    const requestAttrs =
      body === undefined
        ? { endpoint, method, headers }
        : { endpoint, method, body, headers };
    const request = new Request(requestAttrs);

    return request.send();
  }

  static _request(method, endpoint, body, headers) {
    const { GET } = Request.REQUEST_METHODS;
    const requestAttrs =
      method === GET
        ? { endpoint, method, headers }
        : { endpoint, method, body, headers };
    const request = new Request(requestAttrs);

    return request.send();
  }

  _authenticatedHeaders = (headers) => {
    return {
      ...headers,
      Authorization: `Bearer ${this.bearerToken}`,
    };
  };

  _authenticatedRequest(method, endpoint, body, overrideHeaders) {
    const headers = this._authenticatedHeaders(overrideHeaders);

    return Base._request(method, endpoint, body, headers);
  }

  _endpoint(pathname) {
    return this.baseURL + pathname;
  }
}

export default Base;
