import createRequestMock from "test/mocks/create_request_mock";
import { userStub } from "test/stubs";

export default {
  changePassword: {
    valid: (bearerToken, params) => {
      return createRequestMock({
        bearerToken,
        endpoint: "/api/v1/fleet/change_password",
        method: "post",
        params,
        response: {},
      });
    },
  },
  confirmEmailChange: {
    valid: (bearerToken, token) => {
      const endpoint = `/api/v1/fleet/email/change/${token}`;

      return createRequestMock({
        bearerToken,
        endpoint,
        method: "get",
        response: { new_email: "new@email.com" },
      });
    },
  },
  enable: {
    valid: (bearerToken, user, params) => {
      const endpoint = `/api/v1/fleet/users/${user.id}/enable`;

      return createRequestMock({
        bearerToken,
        endpoint,
        method: "post",
        params,
        response: { user: { ...user, ...params } },
      });
    },
  },
  forgotPassword: {
    invalid: (response) => {
      return createRequestMock({
        endpoint: "/api/v1/fleet/forgot_password",
        method: "post",
        response,
        responseStatus: 422,
      });
    },
    valid: () => {
      return createRequestMock({
        endpoint: "/api/v1/fleet/forgot_password",
        method: "post",
        response: { user: userStub },
      });
    },
  },
  loadAll: {
    valid: (bearerToken) => {
      return createRequestMock({
        bearerToken,
        endpoint: "/api/v1/fleet/users?page=0&per_page=100",
        method: "get",
        response: { users: [userStub] },
      });
    },
    validWithParams: (bearerToken) => {
      return createRequestMock({
        bearerToken,
        endpoint:
          "/api/v1/fleet/users?page=3&per_page=100&&order_key=name&order_direction=desc&query=testQuery",
        method: "get",
        response: { users: [userStub] },
      });
    },
  },
  me: {
    valid: (bearerToken) => {
      return createRequestMock({
        bearerToken,
        endpoint: "/api/v1/fleet/me",
        method: "get",
        response: { user: userStub },
      });
    },
  },
  resetPassword: {
    invalid: (password, token, response) => {
      const params = { new_password: password, password_reset_token: token };

      return createRequestMock({
        endpoint: "/api/v1/fleet/reset_password",
        method: "post",
        params,
        response,
        responseStatus: 422,
      });
    },
    valid: (password, token) => {
      const params = { new_password: password, password_reset_token: token };

      return createRequestMock({
        endpoint: "/api/v1/fleet/reset_password",
        method: "post",
        params,
        response: { user: userStub },
      });
    },
  },
  update: {
    valid: (user, params) => {
      return createRequestMock({
        endpoint: `/api/v1/fleet/users/${user.id}`,
        method: "patch",
        params,
        response: { user: userStub },
      });
    },
  },
  updateAdmin: {
    valid: (bearerToken, user, params) => {
      return createRequestMock({
        bearerToken,
        endpoint: `/api/v1/fleet/users/${user.id}/admin`,
        method: "post",
        params,
        response: { user: { ...user, ...params } },
      });
    },
  },
};
