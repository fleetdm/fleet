import createRequestMock from 'test/mocks/create_request_mock';

const errorResponse = {
  message: 'Resource not found',
  errors: [
    {
      name: 'base',
      reason: 'Resource not found',
    },
  ],
};

export default {
  create: {
    valid: (bearerToken, params) => {
      return createRequestMock({
        bearerToken,
        endpoint: '/api/v1/kolide/queries',
        method: 'post',
        params,
        response: { query: params },
        responseStatus: 201,
      });
    },
  },
  destroy: {
    valid: (bearerToken, { id }) => {
      return createRequestMock({
        bearerToken,
        endpoint: `/api/v1/kolide/queries/${id}`,
        method: 'delete',
        response: {},
      });
    },
  },
  load: {
    invalid: (bearerToken, id) => {
      return createRequestMock({
        bearerToken,
        endpoint: `/api/v1/kolide/queries/${id}`,
        method: 'get',
        response: errorResponse,
        responseStatus: 404,
      });
    },
    valid: (bearerToken, id) => {
      return createRequestMock({
        bearerToken,
        endpoint: `/api/v1/kolide/queries/${id}`,
        method: 'get',
        response: { query: { id } },
      });
    },
  },
  loadAll: {
    valid: (bearerToken) => {
      return createRequestMock({
        bearerToken,
        endpoint: '/api/v1/kolide/queries',
        method: 'get',
        response: { queries: [] },
      });
    },
  },
  run: {
    valid: (bearerToken, params) => {
      return createRequestMock({
        bearerToken,
        endpoint: '/api/v1/kolide/queries/run',
        method: 'post',
        params,
        response: { campaign: { id: 1 } },
      });
    },
  },
  update: {
    valid: (bearerToken, query, params) => {
      const endpoint = `/api/v1/kolide/queries/${query.id}`;

      return createRequestMock({
        bearerToken,
        endpoint,
        method: 'patch',
        params,
        response: { query: { ...query, ...params } },
      });
    },
  },
};

