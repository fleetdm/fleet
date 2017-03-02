import createRequestMock from 'test/mocks/create_request_mock';

export default {
  destroy: {
    valid: (bearerToken, host) => {
      return createRequestMock({
        bearerToken,
        endpoint: `/api/v1/kolide/hosts/${host.id}`,
        method: 'delete',
        response: {},
      });
    },
  },
  loadAll: {
    valid: (bearerToken) => {
      return createRequestMock({
        bearerToken,
        endpoint: '/api/v1/kolide/hosts',
        method: 'get',
        response: { hosts: [] },
      });
    },
  },
};
