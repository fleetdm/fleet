import VirtualDatabase from "./db";

import { server } from "./mocks/server.js";

// Establish API mocking before all tests.

beforeAll(() => server.listen());

// Reset any request handlers that we may add during the tests,

// so they don't affect other tests.

afterEach(() => server.resetHandlers());

// Clean up after the tests are finished.

afterAll(() => server.close());

test("initialize db", async () => {
  //const res = await fetch("http://localhost:8080/user");
  const db = await VirtualDatabase.init();
  const res = await db.query("select 1");
  expect(res).toEqual([{ "1": 1 }]);
});
