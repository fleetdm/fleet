import createRequestMock from "test/mocks/create_request_mock";
import { teamScheduledQueryStub } from "test/stubs";

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
        team_id: Number(unformattedParams.team_id),
      };

      return createRequestMock({
        bearerToken,
        endpoint: `/api/v1/fleet/team/${params.team_id}/schedule`,
        method: "post",
        params,
        response: { scheduled: teamScheduledQueryStub },
        responseStatus: 201,
      });
    },
  },
  destroy: {
    valid: (bearerToken, teamScheduledQuery) => {
      return createRequestMock({
        bearerToken,
        endpoint: `/api/v1/fleet/team/${teamScheduledQuery.team_id}/schedule/${teamScheduledQuery.id}`,
        method: "delete",
        response: {},
      });
    },
  },
  loadAll: {
    valid: (bearerToken, teamID) => {
      return createRequestMock({
        bearerToken,
        endpoint: `/api/v1/fleet/team/${teamID}/schedule`,
        method: "get",
        response: { scheduled: [teamScheduledQueryStub] },
      });
    },
  },
  update: {
    valid: (bearerToken, teamScheduledQuery, params) => {
      return createRequestMock({
        bearerToken,
        endpoint: `/api/v1/fleet/team/${teamScheduledQuery.team_id}/schedule/${teamScheduledQuery.id}`,
        method: "patch",
        params,
        response: { scheduled: { ...teamScheduledQuery, ...params } },
      });
    },
  },
};
