import createRequestMock from "test/mocks/create_request_mock";
import { userStub } from "test/stubs";

export default {
  create: {
    valid: (bearerToken = "abc123", params) => {
      return createRequestMock({
        endpoint: "/api/latest/fleet/login",
        method: "post",
        params,
        response: { token: bearerToken, user: userStub },
      });
    },
  },
  destroy: {
    valid: (bearerToken) => {
      return createRequestMock({
        bearerToken,
        endpoint: "/api/latest/fleet/logout",
        method: "post",
        response: {},
      });
    },
  },
};
