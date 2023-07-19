import { jest } from "@jest/globals";

import { server } from "./src/mocks/server.js";

// Establish API mocking before all tests.
beforeAll(() => {
  server.listen();
  // @ts-ignore Not clear why this causes an error??
  global.jest = jest;
});

// Reset any request handlers that we may add during the tests,
// so they don't affect other tests.
afterEach(() => server.resetHandlers());

// Clean up after the tests are finished.
afterAll(() => server.close());
