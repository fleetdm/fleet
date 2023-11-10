import "@testing-library/jest-dom";

import mockServer from "./mock-server";

window.matchMedia =
  window.matchMedia ||
  function matchMedia() {
    return {
      matches: false,
      addEventListener() {
        return null;
      },
    };
  };

// Mock server setup
beforeAll(() => mockServer.listen());
afterEach(() => mockServer.resetHandlers());
afterAll(() => mockServer.close());
