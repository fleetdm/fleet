import createRequestMock from "test/mocks/create_request_mock";
import { packStub } from "test/stubs";

export default {
  addLabel: {
    valid: (bearerToken, packID, labelID) => {
      const endpoint = `/api/v1/fleet/packs/${packID}/labels/${labelID}`;

      return createRequestMock({
        bearerToken,
        endpoint,
        method: "post",
        response: { pack: packStub },
      });
    },
  },
  addQuery: {
    valid: (bearerToken, packID, queryID) => {
      const endpoint = `/api/v1/fleet/packs/${packID}/queries/${queryID}`;

      return createRequestMock({
        bearerToken,
        endpoint,
        method: "post",
        response: { pack: packStub },
      });
    },
  },
  create: {
    valid: (bearerToken, params) => {
      return createRequestMock({
        bearerToken,
        endpoint: "/api/v1/fleet/packs",
        params,
        method: "post",
        response: { pack: params },
        responseStatus: 201,
      });
    },
  },
  destroy: {
    valid: (bearerToken, pack) => {
      return createRequestMock({
        bearerToken,
        endpoint: `/api/v1/fleet/packs/id/${pack.id}`,
        method: "delete",
        response: {},
      });
    },
  },
  update: {
    valid: (bearerToken, pack, params) => {
      return createRequestMock({
        bearerToken,
        endpoint: `/api/v1/fleet/packs/${pack.id}`,
        method: "patch",
        params,
        response: { pack: { ...pack, ...params } },
      });
    },
  },
};
