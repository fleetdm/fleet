import createRequestMock from "test/mocks/create_request_mock";

export default {
  create: {
    valid: (bearerToken, params) => {
      return createRequestMock({
        bearerToken,
        endpoint: "/api/v1/fleet/invites",
        method: "post",
        params,
        response: { invite: params },
      });
    },
  },
  loadAll: {
    valid: (bearerToken) => {
      return createRequestMock({
        bearerToken,
        endpoint: "/api/v1/fleet/invites?page=0&per_page=100",
        method: "get",
        response: { invites: [] },
      });
    },
    validWithParams: (bearerToken) => {
      return createRequestMock({
        bearerToken,
        endpoint:
          "/api/v1/fleet/invites?page=3&per_page=100&&order_key=name&order_direction=desc&query=testQuery",
        method: "get",
        response: { invites: [] },
      });
    },
  },
  destroy: {
    valid: (bearerToken, invite) => {
      return createRequestMock({
        bearerToken,
        endpoint: `/api/v1/fleet/invites/${invite.id}`,
        method: "delete",
        response: {},
      });
    },
  },
};
