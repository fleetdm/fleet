# Frontend mocks

Each `*Mock.ts` file contains one or more default mock objects and their corresponding helper function to create custom mocks by partially overriding the default mock.

## Table of contents
- [Default mocks usage](#default-mocks-usage)
- [Custom mocks usage](#custom-mocks-usage)

## Default mocks

Default mocks are used in expected behavior use cases. Limit the default mock to a single object that can be modified using overrides.

### Example

To test the `ActivityFeed.ts` component, a single default activity is stored in `__mocks__/activityMock` as:

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

This default mock will be returned from calling the `/activities` API using the `defaultActivtyHandler` using `createMockActivity()`.


## Custom mocks

Use the custom mock function to override data to create varying mocked data. If needed, build arrays of mocked data using custom mock overrides.

### Example

Custom mocks are useful when testing larger and/or varying backend responses.

To test the `ActivityFeed.ts` component, we modified the single default mock activity stored in `__mocks__/activityMock` by partially overriding the mock with a helper function.

The helper function
`createMockActivity({ id: 2, actor_full_name: "Gabe" })` will return modifications to the `DEFAULT_ACTIVITY_MOCK` to override the `id` and `actor_full_name` keys only.

This may be useful for a default handler like `defaultActivityHandler` or a custom handler like `activityHandler9Activities` to return several activities varying from the default activity.

