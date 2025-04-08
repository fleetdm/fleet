import endpoints from "utilities/endpoints";
import {
  ActivityType,
  IActivity,
  IHostPastActivity,
  IHostUpcomingActivity,
} from "interfaces/activity";
import sendRequest from "services";
import { buildQueryStringFromParams } from "utilities/url";

const DEFAULT_PAGE = 0;
const DEFAULT_PAGE_SIZE = 8;
const ORDER_KEY = "created_at";
const ORDER_DIRECTION = "desc";

export interface IActivitiesResponse {
  activities: IActivity[] | null;
  meta: {
    has_next_results: boolean;
    has_previous_results: boolean;
  };
}

export interface IHostPastActivitiesResponse {
  activities: IHostPastActivity[] | null;
  meta: {
    has_next_results: boolean;
    has_previous_results: boolean;
  };
}

export interface IHostUpcomingActivitiesResponse {
  count: number;
  activities: IHostUpcomingActivity[] | null;
  meta: {
    has_next_results: boolean;
    has_previous_results: boolean;
  };
}

export default {
  loadNext: (
    page = DEFAULT_PAGE,
    perPage = DEFAULT_PAGE_SIZE
  ): Promise<IActivitiesResponse> => {
    const { ACTIVITIES } = endpoints;

    const queryParams = {
      page,
      per_page: perPage,
      order_key: ORDER_KEY,
      order_direction: ORDER_DIRECTION,
    };

    const queryString = buildQueryStringFromParams(queryParams);

    const path = `${ACTIVITIES}?${queryString}`;
    // TODO - restore real API call

    // return sendRequest("GET", path);
    return sendRequest("GET", path).then((response) => {
      const fakeBase = {
        created_at: "2022-11-03T17:22:14Z",
        id: 1,
        actor_full_name: "Test User",
        actor_id: 1,
        actor_gravatar: "",
        actor_email: "test@example.com",
        fleet_initiated: false,
        // type: ActivityType.ConfiguredMSEntraConditionalAccess,
        // type: ActivityType.DeletedMSEntraConditionalAccess,
        // type: ActivityType.EnabledConditionalAccessAutomations,
        // type: ActivityType.DisabledConditionalAccessAutomations,
        // details: {
        //   team_name: "Test Team",
        // },
      };
      const configured: IActivity = {
        ...fakeBase,
        type: ActivityType.ConfiguredMSEntraConditionalAccess,
      };
      const deleted: IActivity = {
        ...fakeBase,
        type: ActivityType.DeletedMSEntraConditionalAccess,
      };
      const enabledTeam: IActivity = {
        ...fakeBase,
        type: ActivityType.EnabledConditionalAccessAutomations,
        details: {
          team_name: "Test Team",
        },
      };
      const enabledNoTeam: IActivity = {
        ...fakeBase,
        type: ActivityType.EnabledConditionalAccessAutomations,
      };
      const disabledTeam: IActivity = {
        ...fakeBase,
        type: ActivityType.DisabledConditionalAccessAutomations,
        details: {
          team_name: "Test Team",
        },
      };
      const disabledNoTeam: IActivity = {
        ...fakeBase,
        type: ActivityType.DisabledConditionalAccessAutomations,
      };
      return {
        meta: response.meta,
        activities: [
          deleted,
          disabledTeam,
          enabledTeam,
          disabledNoTeam,
          enabledNoTeam,
          configured,
          ...response.activities,
        ],
      };
    });
  },

  getHostPastActivities: (
    id: number,
    page = DEFAULT_PAGE,
    perPage = DEFAULT_PAGE_SIZE
  ): Promise<IHostPastActivitiesResponse> => {
    const { HOST_PAST_ACTIVITIES } = endpoints;

    const queryParams = {
      page,
      per_page: perPage,
    };

    const queryString = buildQueryStringFromParams(queryParams);

    const path = `${HOST_PAST_ACTIVITIES(id)}?${queryString}`;

    return sendRequest("GET", path);
  },

  getHostUpcomingActivities: (
    id: number,
    page = DEFAULT_PAGE,
    perPage = DEFAULT_PAGE_SIZE
  ): Promise<IHostUpcomingActivitiesResponse> => {
    const { HOST_UPCOMING_ACTIVITIES } = endpoints;

    const queryParams = {
      page,
      per_page: perPage,
    };

    const queryString = buildQueryStringFromParams(queryParams);

    const path = `${HOST_UPCOMING_ACTIVITIES(id)}?${queryString}`;

    return sendRequest("GET", path);
  },

  cancelHostActivity: (hostId: number, uuid: string) => {
    const { HOST_CANCEL_ACTIVITY } = endpoints;
    return sendRequest("DELETE", HOST_CANCEL_ACTIVITY(hostId, uuid));
  },
};
