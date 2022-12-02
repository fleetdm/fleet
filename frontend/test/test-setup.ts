import "@testing-library/jest-dom";

import mockServer from "./mock-server";

// Mock server setup
beforeAll(() => mockServer.listen());
afterEach(() => mockServer.resetHandlers());
afterAll(() => mockServer.close());
