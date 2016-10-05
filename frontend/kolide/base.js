import fetch from 'isomorphic-fetch';
import local from '../utilities/local';

const REQUEST_METHODS = {
  GET: 'GET',
  PATCH: 'PATCH',
  POST: 'POST',
};

class Base {
  constructor () {
    const { origin } = global.window.location;

    this.baseURL = `${origin}/api`;
    this.bearerToken = local.getItem('auth_token');
  }

  endpoint (pathname) {
    return this.baseURL + pathname;
  }

  setBearerToken (bearerToken) {
    this.bearerToken = bearerToken;
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

  post (endpoint, body = {}, overrideHeaders = {}) {
    const { POST } = REQUEST_METHODS;

    return this._request(POST, endpoint, body, overrideHeaders);
  }

  _authenticatedRequest(method, endpoint, body, overrideHeaders) {
    const headers = {
      ...overrideHeaders,
      Authorization: `Bearer ${this.bearerToken}`,
    };

    return this._request(method, endpoint, body, headers);
  }

  _request (method, endpoint, body, overrideHeaders) {
    const credentials = 'same-origin';
    const { GET } = REQUEST_METHODS;
    const headers = {
      Accept: 'application/json',
      'Content-Type': 'application/json',
      ...overrideHeaders,
    };
    const requestAttrs = method === GET
      ? { credentials, method, headers }
      : { credentials, method, body, headers };

    return fetch(endpoint, requestAttrs)
      .then(response => {
        return response.json()
          .then(jsonResponse => {
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
}

export default Base;

