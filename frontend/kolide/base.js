import fetch from 'isomorphic-fetch';

import local from '../utilities/local';

const REQUEST_METHODS = {
  DELETE: 'DELETE',
  GET: 'GET',
  PATCH: 'PATCH',
  POST: 'POST',
};

class Base {
  constructor () {
    const { host, origin } = global.window.location;

    this.baseURL = `${origin}/api`;
    this.websocketBaseURL = `wss://${host}/api`;
    this.bearerToken = local.getItem('auth_token');
  }

  static _request (method, endpoint, body, overrideHeaders) {
    const credentials = 'same-origin';
    const { DELETE, GET } = REQUEST_METHODS;
    const headers = {
      Accept: 'application/json',
      'Content-Type': 'application/json',
      ...overrideHeaders,
    };
    const requestAttrs = method === GET
      ? { credentials, method, headers }
      : { credentials, method, body, headers };

    return fetch(endpoint, requestAttrs)
      .then((response) => {
        if (method === DELETE) return false;

        return response.json()
          .then((jsonResponse) => {
            if (response.ok) {
              return jsonResponse;
            }

            const error = new Error(response.statusText);
            error.response = jsonResponse;
            error.message = jsonResponse;
            error.error = jsonResponse.error;

            throw error;
          });
      });
  }

  static post (endpoint, body = {}, overrideHeaders = {}) {
    const { POST } = REQUEST_METHODS;

    return Base._request(POST, endpoint, body, overrideHeaders);
  }

  endpoint (pathname) {
    return this.baseURL + pathname;
  }

  setBearerToken (bearerToken) {
    this.bearerToken = bearerToken;
  }

  authenticatedDelete (endpoint, overrideHeaders = {}) {
    const { DELETE } = REQUEST_METHODS;

    return this._authenticatedRequest(DELETE, endpoint, {}, overrideHeaders);
  }

  authenticatedGet (endpoint, overrideHeaders = {}) {
    const { GET } = REQUEST_METHODS;

    return this._authenticatedRequest(GET, endpoint, {}, overrideHeaders);
  }

  authenticatedPatch (endpoint, body = {}, overrideHeaders = {}) {
    const { PATCH } = REQUEST_METHODS;

    return this._authenticatedRequest(PATCH, endpoint, body, overrideHeaders);
  }

  authenticatedPost (endpoint, body = {}, overrideHeaders = {}) {
    const { POST } = REQUEST_METHODS;

    return this._authenticatedRequest(POST, endpoint, body, overrideHeaders);
  }

  _authenticatedHeaders = (headers) => {
    return {
      ...headers,
      Authorization: `Bearer ${this.bearerToken}`,
    };
  }

  _authenticatedRequest(method, endpoint, body, overrideHeaders) {
    const headers = this._authenticatedHeaders(overrideHeaders);

    return Base._request(method, endpoint, body, headers);
  }
}

export default Base;

