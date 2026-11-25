import { setupServer } from "msw/node";

const mockServer = setupServer();

// Registered here, not in test-setup.ts, so only the test files that import
// the server pay for loading and starting it. Most never make a request.
beforeAll(() => mockServer.listen());
afterEach(() => mockServer.resetHandlers());
afterAll(() => mockServer.close());

export default mockServer;
