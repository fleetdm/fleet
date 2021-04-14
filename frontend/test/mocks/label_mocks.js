import createRequestMock from "test/mocks/create_request_mock";

export default {
  create: {
    valid: (bearerToken, params) => {
      return createRequestMock({
        bearerToken,
        endpoint: "/api/v1/fleet/labels",
        method: "post",
        response: { label: { ...params, display_text: params.name } },
        responseStatus: 201,
      });
    },
  },
  destroy: {
    valid: (bearerToken, label) => {
      return createRequestMock({
        bearerToken,
        endpoint: `/api/v1/fleet/labels/id/${label.id}`,
        method: "delete",
        response: {},
      });
    },
  },
  update: {
    valid: (bearerToken, label, params) => {
      return createRequestMock({
        bearerToken,
        endpoint: `/api/v1/fleet/labels/${label.id}`,
        method: "patch",
        response: { label: { ...label, ...params } },
      });
    },
  },
};
