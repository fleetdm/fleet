import "@testing-library/jest-dom";

import mockServer from "./mock-server";

window.CSS.supports = jest.fn();

// Mock server setup
beforeAll(() => mockServer.listen());
afterEach(() => mockServer.resetHandlers());
afterAll(() => mockServer.close());
