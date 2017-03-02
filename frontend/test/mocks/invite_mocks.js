import createRequestMock from 'test/mocks/create_request_mock';

export default {
  create: {
    valid: (bearerToken, params) => {
      return createRequestMock({
        bearerToken,
        endpoint: '/api/v1/kolide/invites',
        method: 'post',
        params,
        response: { invite: params },
      });
    },
  },
  loadAll: {
    valid: (bearerToken) => {
      return createRequestMock({
        bearerToken,
        endpoint: '/api/v1/kolide/invites',
        method: 'get',
        response: { invites: [] },
      });
    },
  },
  destroy: {
    valid: (bearerToken, invite) => {
      return createRequestMock({
        bearerToken,
        endpoint: `/api/v1/kolide/invites/${invite.id}`,
        method: 'delete',
        response: {},
      });
    },
  },
};

