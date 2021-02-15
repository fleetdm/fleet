import createRequestMock from 'test/mocks/create_request_mock';

export default {
  destroy: {
    valid: (bearerToken, host) => {
      return createRequestMock({
        bearerToken,
        endpoint: `/api/v1/fleet/hosts/${host.id}`,
        method: 'delete',
        response: {},
      });
    },
  },
  loadAll: {
    valid: (bearerToken) => {
      return createRequestMock({
        bearerToken,
        endpoint: '/api/v1/fleet/hosts?page=0&per_page=100&order_key=hostname',
        method: 'get',
        response: { hosts: [] },
      });
    },
  },
};
