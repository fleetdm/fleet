import createRequestMock from "test/mocks/create_request_mock";

export default {
  loadAll: {
    valid: (bearerToken) => {
      return createRequestMock({
        bearerToken,
        endpoint: "/api/latest/fleet/config",
        method: "get",
        response: { config: { name: "Fleet" } },
      });
    },
  },
  update: {
    valid: (bearerToken, params) => {
      return createRequestMock({
        bearerToken,
        endpoint: "/api/latest/fleet/config",
        method: "patch",
        params,
        response: {},
      });
    },
  },
};
