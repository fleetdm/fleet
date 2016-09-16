import config from '../config';

class Base {
  constructor () {
    this.baseURL = this.setBaseURL();
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

  post(endpoint, body = {}, overrideHeaders = {}) {
    return this._request('POST', endpoint, body, overrideHeaders);
  }

  _request (method, endpoint, body, overrideHeaders) {
    const headers = {
      Accept: 'application/json',
      'Content-Type': 'application/json',
    };

    return fetch(endpoint, {
      method,
      headers: {
        ...headers,
        ...overrideHeaders
      },
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

