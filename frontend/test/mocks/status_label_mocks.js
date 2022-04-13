import createRequestMock from "test/mocks/create_request_mock";

export default {
  getCounts: {
    valid: (bearerToken) => {
      return createRequestMock({
        bearerToken,
        endpoint: "/api/latest/fleet/host_summary",
        method: "get",
        response: { online_count: 1, offline_count: 23, mia_count: 2 },
      });
    },
  },
};
