import fetch from 'isomorphic-fetch';
import local from '../utilities/local';

class Base {
  constructor () {
    const { origin } = global.window.location;

    this.baseURL = `${origin}/api`;
    this.bearerToken = local.getItem('auth_token');
  }

  setBearerToken (bearerToken) {
    this.bearerToken = bearerToken;
  }

  authenticatedGet (endpoint, overrideHeaders = {}) {
    return this._authenticatedRequest('GET', endpoint, {}, overrideHeaders);
  }

  authenticatedPost (endpoint, body = {}, overrideHeaders = {}) {
    return this._authenticatedRequest('POST', endpoint, body, overrideHeaders);
  }

  post (endpoint, body = {}, overrideHeaders = {}) {
    return this._request('POST', endpoint, body, overrideHeaders);
  }

  _authenticatedRequest(method, endpoint, body, overrideHeaders) {
    const headers = {
      ...overrideHeaders,
      Authorization: `Bearer ${this.bearerToken}`,
    };

    return this._request(method, endpoint, body, headers);
  }

  _request (method, endpoint, body, overrideHeaders) {
    const headers = {
      Accept: 'application/json',
      'Content-Type': 'application/json',
      ...overrideHeaders,
    };

    return fetch(endpoint, {
      credentials: 'same-origin',
      method,
      headers,
      body,
    })
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

