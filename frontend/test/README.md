# Fleet UI tests

The test directory contains the jest configuration, test setup, request handlers, mock server definition,  testing utilities, and entity stubs (deprecated and will be replaced by mocks in `frontend/__mocks__`) for use in test files throughout the application. The test files for components and app functions are located in the same directory as the files they test.

<!-- 
TODO

The default export from the test directory includes mock server with default handlers, custom handlers, testing stubs, and testing utilities like custom renderers. -->

## Table of contents
- [Jest configuration](#jest-configuration)
- [Test setup](#test-setup)
- [Request handlers and their setup](#request-handlers-and-their-setup)
- [Testing utilities](#testing-utilities)
- [Entity stubs (deprecated)](#entity-stubs-deprecated)
- [Related links](#related-links)

## Jest configuration

This is where the jest configuration is located. Refer to [Jest's official documentation](https://jestjs.io/docs/configuration).
## Test setup

This file configures the testing environment for every test file.

## Request handlers and their setup

Request handlers are defined within the `handlers` directory and return [mocked data](../__mocks__/README.md). The handlers directory will naturally grow as more tests need them. We use [mock service worker](https://mswjs.io/docs/api/rest) to define all request handlers.

The mock server in [mock-server.ts](./mock-server.ts) starts with no handlers. Every handler a test needs must be setup inline within the component's test suite (`frontend/**/ComponentName.tests.tsx`). For example, we would setup the handler `activityHandler9Activities` inline using `mockServer.use(activityHandler9Activities);`. If every test in the file needs the same handler, call `mockServer.use` in a `beforeEach`.

The mock server only runs in test files that import it. Requests from a test file that doesn't import it aren't mocked.

## Testing utilities

We use various utility functions to write our tests.

## Testing stubs `Deprecated`

Testing stubs are still being used in a handful of old tests. We are no longer following this pattern of adding data to testing stubs. Rather, we are building stubs as mocks located in the `frontend/__mocks__` directory.

## Related links

Check out how we [mock data](../__mocks__/README.md) used for unit and integration tests.

Follow [this guide](../../docs/Contributing/getting-started/testing-and-local-development.md) to run tests locally.

Visit the frontend [overview of Fleet UI testing](../docs/Contributing/guides/ui/fleet-ui-testing.md) for more information on our testing strategy, philosophies, and tools.