# Frontend mocks

Each `__mocks___/*Mock.ts` file contains one or more default mock objects and their corresponding helper function to partially override the default mock creating custom mocks.

## Table of contents
- [Default mocks usage](#default-mocks-usage)
  -[Example](#example)
- [Custom mocks usage](#custom-mocks-usage)
  -[Global handlers vs. inline handlers](#global-handlers-vs-inline-handlers)
  -[Examples](#examples)
- [Related links](#related-links)

## Default mocks

Default mocks are used in default behavior use cases. We limit the default mock to a single object that can be modified using overrides.

### Example

To test the `ActivityFeed.ts` component, a single default activity is stored in `__mocks__/activityMock.ts` as:

```
const DEFAULT_ACTIVITY_MOCK: IActivity = {
  created_at: "2022-11-03T17:22:14Z",
  id: 1,
  actor_full_name: "Rachel",
  actor_id: 1,
  actor_gravatar: "",
  actor_email: "rachel@fleetdm.com",
  type: ActivityType.EditedAgentOptions,
};
```

This default mock will be returned from calling the `/activities` API in `test/handlers/activity-handlers.ts` using the `defaultActivityHandler` function. Within `defaultActivityHandler`, we use the mock function `createMockActivity()` with no arguments to return the `DEFAULT_ACTIVITY_MOCK`. Contrastingly, we can add arguments to create custom mocks.


## Custom mocks

Custom mocks are useful when testing larger and/or varying backend responses.

Use a mock function with arguments to override default mock data to create varying mocked data. If needed, build arrays or more complex objects from mocked data using custom mock overrides.

### Global handlers vs. inline handlers

Custom handlers can be stored in a global handlers directory `test/handlers` along with default handlers, or inline within a component's test suite (`frontend/**/ComponentName.tests.tsx`).

#### Examples

1. Global handler `test/handlers/activity-handlers.ts`

To test the `ActivityFeed.ts` component, we modified the single default mock activity stored in `__mocks__/activityMock.ts` by partially overriding the mock with a helper function.

Within `test/handlers/activity-handlers.ts`, the helper function
`createMockActivity({ id: 2, actor_full_name: "Gabe" })` will return modifications to the `DEFAULT_ACTIVITY_MOCK` to override the `id` and `actor_full_name` keys only.

This may be useful for a default handler like `defaultActivityHandler` or a custom handler like `activityHandler9Activities` to return several activities varying from the default activity.

2. Inline handler `ActivityItem.tests.tsx`

To test specific activities created by `ActivityItem.tsx`, we created custom mocks needed for each activity needed to be tested using arguments in our helper function `createMockActivity()`.

For example, in `ActivityItem.tests.tsx`, we want to ensure the activity feed shows when a user runs a live query. We can return mocked data for this inline using `const activity = createMockActivity({ type: ActivityType.LiveQuery });`.

### Related links

Check out the [frontend test directory](../test/README.md) for information about our unit and integration testing layers.

Follow [this guide](../../docs/Contributing/Testing-and-local-development.md) to run tests locally.


