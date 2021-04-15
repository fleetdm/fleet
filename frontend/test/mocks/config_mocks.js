import createRequestMock from "test/mocks/create_request_mock";

export default {
  loadAll: {
    valid: (bearerToken) => {
      return createRequestMock({
        bearerToken,
        endpoint: "/api/v1/fleet/config",
        method: "get",
        response: { config: { name: "Kolide" } },
      });
    },
  },
  update: {
    valid: (bearerToken, params) => {
      return createRequestMock({
        bearerToken,
        endpoint: "/api/v1/fleet/config",
        method: "patch",
        params,
        response: {},
      });
    },
  },
};
