import createRequestMock from "test/mocks/create_request_mock";
import { scheduledQueryStub } from "test/stubs";

export default {
  create: {
    valid: (bearerToken, unformattedParams) => {
      const params = {
        interval: Number(unformattedParams.interval),
        pack_id: Number(unformattedParams.pack_id),
        platform: unformattedParams.platform,
        query_id: Number(unformattedParams.query_id),
        removed: true,
        snapshot: false,
        shard: Number(unformattedParams.shard),
      };

      return createRequestMock({
        bearerToken,
        endpoint: "/api/v1/fleet/schedule",
        method: "post",
        params,
        response: { scheduled: scheduledQueryStub },
        responseStatus: 201,
      });
    },
  },
  destroy: {
    valid: (bearerToken, scheduledQuery) => {
      return createRequestMock({
        bearerToken,
        endpoint: `/api/v1/fleet/schedule/${scheduledQuery.id}`,
        method: "delete",
        response: {},
      });
    },
  },
  loadAll: {
    valid: (bearerToken, pack) => {
      return createRequestMock({
        bearerToken,
        endpoint: `/api/v1/fleet/packs/${pack.id}/scheduled`,
        method: "get",
        response: { scheduled: [scheduledQueryStub] },
      });
    },
  },
  update: {
    valid: (bearerToken, scheduledQuery, params) => {
      return createRequestMock({
        bearerToken,
        endpoint: `/api/v1/fleet/schedule/${scheduledQuery.id}`,
        method: "patch",
        params,
        response: { scheduled: { ...scheduledQuery, ...params } },
      });
    },
  },
};
