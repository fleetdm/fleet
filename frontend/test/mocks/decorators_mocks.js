import createRequestMock from 'test/mocks/create_request_mock';


export default {
  loadAll: {
    valid: (bearerToken) => {
      return createRequestMock({
        bearerToken,
        endpoint: '/api/v1/kolide/decorators',
        method: 'get',
        response: { decorators: [] },
      });
    },
  },
  create: {
    valid: (bearerToken, params) => {
      const req = { payload: params };
      return createRequestMock({
        bearerToken,
        endpoint: '/api/v1/kolide/decorators',
        method: 'post',
        req,
        response: { decorator: params },
        responseStatus: 201,
      });
    },
  },
  destroy: {
    valid: (bearerToken, { id }) => {
      return createRequestMock({
        bearerToken,
        endpoint: `/api/v1/kolide/decorators/${id}`,
        method: 'delete',
        response: {},
      });
    },
  },
};
