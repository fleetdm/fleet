import "@testing-library/jest-dom";
import mockServer from "./mock-server";

// Needed for testing react-tooltip-5
window.CSS.supports = jest.fn();

// JSDOM does not implement matchMedia, which useIsMobileWidth (DeviceUserPage)
// calls unguarded on mount. Default to the desktop breakpoint; tests that need
// mobile can override.
Object.defineProperty(window, "matchMedia", {
  writable: true,
  value: jest.fn().mockImplementation((query) => ({
    matches: false,
    media: query,
    addEventListener: jest.fn(),
    removeEventListener: jest.fn(),
    addListener: jest.fn(), // for older APIs
    removeListener: jest.fn(),
    onchange: null,
    dispatchEvent: jest.fn(),
  })),
});
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
