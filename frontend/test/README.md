# Fleet tests

The test directory contains helper functions, request mocks, and entity stubs for use in test files throughout the application. The test files for components and app functions are located in the same directory as the files they test.

The default export from the test directory includes mock server with default handlers, custom handlers, testing stubs, and testing utilities like custom renderers.

## Table of contents
- [Test setup](#test-setup)
- [Custom handlers](#custom-handlers)
- [Testing stubs](#testing-stubs)
- [Testing utilities](#testing-utilities)
- [Related links](#related-links)

## Test setup


As outlined in `test-setup.ts`, the mock server will automatically serve all default handlers at the beginning of each test suite. Between tests,  handlers reset to the default handlers. At the end of each test, the mock server will close.

The mock server `mock-server.ts` will serve the default handlers outlined in `default-handlers.ts` which are imported from the `handlers` directory.


## Custom handlers

In addition to default handlers, custom handlers are stored within the `handlers` directory. Custom handlers are modifications to the default handlers that will pass back custom API data that are likely only used in one or two tests.

The handlers directory will naturally grow with more default and custom handlers required for more unit and integration tests.

## Testing stubs `Deprecated`

Testing stubs are still being used in a handful of old tests. We are no longer following this pattern of adding data to testing stubs. Rather, we are building default handlers for returned mocks and using custom handlers to return modifications to these mocks.

## Testing utilities

TODO:
