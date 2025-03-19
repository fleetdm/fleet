import { http, HttpResponse } from "msw";

import createMockActivity from "__mocks__/activityMock";
import { baseUrl } from "test/test-utils";

export const defaultActivityHandler = http.get(baseUrl("/activities"), () => {
  return HttpResponse.json({
    activities: [
      createMockActivity(),
      createMockActivity({ id: 2, actor_full_name: "Test User 2" }),
      createMockActivity({ id: 3, actor_full_name: "Test User 3" }),
    ],
    meta: {
      has_next_results: false,
      has_previous_results: false,
    },
  });
});

export const activityHandlerHasMoreActivities = http.get(
  baseUrl("/activities"),
  () => {
    return HttpResponse.json({
      activities: [
        createMockActivity(),
        createMockActivity({ id: 2 }),
        createMockActivity({ id: 3 }),
        createMockActivity({ id: 4 }),
        createMockActivity({ id: 5 }),
        createMockActivity({ id: 6 }),
        createMockActivity({ id: 7 }),
        createMockActivity({ id: 8 }),
        createMockActivity({ id: 9 }),
        createMockActivity({ id: 10 }),
        createMockActivity({ id: 11 }),
      ],
      meta: {
        has_next_results: true,
        has_previous_results: false,
      },
    });
  }
);

export const activityHandlerHasPreviousActivities = http.get(
  baseUrl("/activities"),
  () => {
    return HttpResponse.json({
      activities: [
        createMockActivity(),
        createMockActivity({ id: 2 }),
        createMockActivity({ id: 3 }),
      ],
      meta: {
        has_next_results: false,
        has_previous_results: true,
      },
    });
  }
);
