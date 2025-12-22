import "@testing-library/jest-dom";
import mockServer from "./mock-server";

// Needed for testing react-tooltip-5
window.CSS.supports = jest.fn();
global.ResizeObserver = jest.fn().mockImplementation(() => ({
  observe: jest.fn(),
  unobserve: jest.fn(),
  disconnect: jest.fn(),
}));

// Mock server setup
beforeAll(() => mockServer.listen());
afterEach(() => mockServer.resetHandlers());
afterAll(() => mockServer.close());

// suppress the opacity console warnings for react-tooltip. The code for assigning the
// opacity is correct but there is still an unnecessary warning in the console when
// the jest tests are run. This may be react-tooltip and JSdom not playing well together.
beforeAll(() => {
  const originalConsoleWarning = console.warn;
  console.warn = (...args) => {
    if (
      args[0]?.includes("[react-tooltip]") &&
      args[0]?.includes("is not a valid `opacity`")
    ) {
      return;
    }
    originalConsoleWarning(...args);
  };
});
