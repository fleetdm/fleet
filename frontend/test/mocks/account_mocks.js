import createRequestMock from "test/mocks/create_request_mock";
import helpers from "fleet/helpers";

export default {
  create: {
    valid: (unformattedParams) => {
      const params = helpers.setupData(unformattedParams);

      return createRequestMock({
        endpoint: "/api/v1/setup",
        method: "post",
        params,
        response: {},
      });
    },
  },
};
