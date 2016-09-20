import fetch from 'isomorphic-fetch';

import config from '../config';
import local from '../utilities/local';

class Base {
  constructor () {
    this.baseURL = this.setBaseURL();
    this.bearerToken = local.getItem('auth_token');
  }

  setBaseURL () {
    const {
      settings: { env },
      environments: { development },
    } = config;

    if (env === development) {
      return 'http://localhost:8080/api';
    }

    throw new Error(`API base URL is not configured for environment: ${env}`);
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

