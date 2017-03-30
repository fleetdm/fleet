import createRequestMock from 'test/mocks/create_request_mock';

export default {
  loadAll: {
    valid: (bearerToken) => {
      return createRequestMock({
        bearerToken,
        endpoint: '/api/v1/kolide/options',
        method: 'get',
        response: { options: [] },
      });
    },
  },
  update: {
    valid: (bearerToken, params) => {
      return createRequestMock({
        bearerToken,
        endpoint: '/api/v1/kolide/options',
        method: 'patch',
        params: { options: params },
        response: { options: params },
      });
    },
  },
  reset: {
    valid: (bearerToken) => {
      return createRequestMock({
        bearerToken,
        endpoint: '/api/v1/kolide/options/reset',
        method: 'get',
        response: { options: [] },
      });
    },
  },
};
