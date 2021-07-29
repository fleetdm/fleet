import createRequestMock from "test/mocks/create_request_mock";
import { globalScheduledQueryStub } from "test/stubs";

export default {
  create: {
    valid: (bearerToken, unformattedParams) => {
      const params = {
        interval: Number(unformattedParams.interval),
        platform: unformattedParams.platform,
        query_id: Number(unformattedParams.query_id),
        removed: true,
        snapshot: false,
        shard: Number(unformattedParams.shard),
      };

      return createRequestMock({
        bearerToken,
        endpoint: "/api/v1/fleet/global/schedule",
        method: "post",
        params,
        response: { scheduled: globalScheduledQueryStub },
        responseStatus: 201,
      });
    },
  },
  destroy: {
    valid: (bearerToken, globalScheduledQuery) => {
      return createRequestMock({
        bearerToken,
        endpoint: `/api/v1/fleet/global/schedule/${globalScheduledQuery.id}`,
        method: "delete",
        response: {},
      });
    },
  },
  loadAll: {
    valid: (bearerToken) => {
      return createRequestMock({
        bearerToken,
        endpoint: "/api/v1/fleet/global/schedule",
        method: "get",
        response: { scheduled: [globalScheduledQueryStub] },
      });
    },
  },
  update: {
    valid: (bearerToken, globalScheduledQuery, params) => {
      return createRequestMock({
        bearerToken,
        endpoint: `/api/v1/fleet/global/schedule/${globalScheduledQuery.id}`,
        method: "patch",
        params,
        response: { scheduled: { ...globalScheduledQuery, ...params } },
      });
    },
  },
};
