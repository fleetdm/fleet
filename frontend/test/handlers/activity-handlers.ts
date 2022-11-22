import { rest } from "msw";

import createMockActivity from "__mocks__/activityMock";
import { baseUrl } from "test/test-utils";

export const defaultActivityHandler = rest.get(
  baseUrl("/activities"),
  (req, res, context) => {
    return res(
      context.json({
        activities: [
          createMockActivity(),
          createMockActivity({ id: 2, actor_full_name: "Gabe" }),
          createMockActivity({ id: 3, actor_full_name: "Luke" }),
        ],
      })
    );
  }
);

export const activityHandler9Activities = rest.get(
  baseUrl("/activities"),
  (req, res, context) => {
    return res(
      context.json({
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
        ],
      })
    );
  }
);
