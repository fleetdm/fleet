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

Default mocks are simple to work with objects. We limit the default mock to a single object that can be modified with the helper function as needed using overrides.

The default mock object is returned by calling the helper function with no arguments.

### Example

A single default activity is defined in `__mocks__/activityMock.ts` as:

```
const DEFAULT_ACTIVITY_MOCK: IActivity = {
  created_at: "2022-11-03T17:22:14Z",
  id: 1,
  actor_full_name: "Test",
  actor_id: 1,
  actor_gravatar: "",
  actor_email: "test@example.com",
  type: ActivityType.EditedAgentOptions,
};
```

To return this default object, call its helper function `createActivityMock()` with no arguments.

## Custom mocks

Custom mocks are useful when we need a mock object with specific data.

Use the helper function with arguments to override the default mock data with the specific data you need.

#### Example

`createMockActivity({ id: 2, actor_full_name: "Gabe" })` will return modifications to the `DEFAULT_ACTIVITY_MOCK` to override the `id` and `actor_full_name` keys only.

### Related links

Check out the [frontend test directory](../test/README.md) for information about our unit and integration testing layers. We use default mocks and custom mocks when [mocking server requests](../test/README.md#server-handlers).

Follow [this guide](../../docs/Contributing/getting-started/testing-and-local-development.md) to run tests locally.

Visit the frontend [overview of Fleet UI testing](../docs/Contributing/guides/ui/fleet-ui-testing.md) for more information on our testing strategy, philosophies, and tools.


