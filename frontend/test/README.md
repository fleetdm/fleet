# Fleet tests

The test directory contains request handlers, TODO, and entity stubs for use in test files throughout the application. The test files for components and app functions are located in the same directory as the files they test.

TODO

The default export from the test directory includes mock server with default handlers, custom handlers, testing stubs, and testing utilities like custom renderers.

## Table of contents
- [Test setup](#test-setup)
- [Server handlers](#server-handlers)
- [Testing stubs](#testing-stubs)
- [Testing utilities](#testing-utilities)
- [Related links](#related-links)

## Test setup

As outlined in `test-setup.ts`, the mock server will automatically serve all default handlers at the beginning of each test suite. Between tests,  handlers reset to the default handlers. At the end of each test suite, the mock server will close.

The mock server `mock-server.ts` will serve the default handlers outlined in `default-handlers.ts` which are imported from the `handlers` directory.


## Server handlers

Default handlers and global custom handlers are stored within the `handlers` directory. The default handler and custom handlers located within `handlers` return [mocked data](../__mocks__/README.md) that is used in a broader scope one or more tests suites. The handlers directory will naturally grow with more default and custom handlers required for more unit and integration tests.

Contrastingly, narrow scope handlers returning [custom mocks](../__mocks__/README.md#custom-mocks) can be more readable and maintainable written [inline](../__mocks__/README.md#global-handlers-vs-inline-handlers).

## Testing stubs `Deprecated`

Testing stubs are still being used in a handful of old tests. We are no longer following this pattern of adding data to testing stubs. Rather, we are building default handlers for returned mocks and using custom handlers to return modifications to these mocks.

## Testing utilities

TODO:

## Related links

Check out how we [mock data](../__mocks__/README.md) used for unit and integration tests.

Follow [our guide](../../docs/Contributing/Testing-and-local-development.md) to run frontend tests locally.