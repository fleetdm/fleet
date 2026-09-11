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
// jsdom has no ResizeObserver, so this is a polyfill, not a test double, and it deliberately avoids jest.fn(). A suite
// that needs a spy can swap in its own and restore it (see HostsEnrolledCard.tests.tsx).
const noop = () => undefined;
global.ResizeObserver = class {
  observe = noop;
  unobserve = noop;
  disconnect = noop;
};

// Mock server setup
beforeAll(() => mockServer.listen());
afterEach(() => mockServer.resetHandlers());
afterAll(() => mockServer.close());

// Known noisy warnings that don't indicate real bugs in Fleet code. Filtering
// them keeps a genuine console.error/warn visible when a test actually fails,
// instead of it being buried under thousands of unrelated lines.
const SUPPRESSED_WARNINGS = [
  // react-tooltip and jsdom disagree on a computed opacity; the app's usage is correct.
  /\[react-tooltip\].*is not a valid `opacity`/,
  // Async state updates inside third-party UI internals (react-select, Radix) and
  // many existing components fire outside RTL's act() boundary. Enforcing
  // act-wrapping across the whole app is a separate, larger cleanup effort.
  /An update to .+ inside a test was not wrapped in act\(\.\.\.\)/,
  // A dependency still calls the old, renamed React lifecycle methods internally.
  /componentWill(Mount|ReceiveProps) has been renamed/,
];

// Trimming node_modules frames out of the stack these produce happens in
// frontend/test/trimConsoleStackReporter.js (registered as a jest reporter) —
// that runs in Jest's own process, unlike this file, which runs inside each
// test file's sandboxed module registry and can't reach Jest's console output
// pipeline directly.
function suppressKnownWarnings(original: typeof console.warn) {
  return (...args: Parameters<typeof console.warn>) => {
    const [message] = args;
    if (
      typeof message === "string" &&
      SUPPRESSED_WARNINGS.some((pattern) => pattern.test(message))
    ) {
      return;
    }
    original(...args);
  };
}

beforeAll(() => {
  console.warn = suppressKnownWarnings(console.warn);
  console.error = suppressKnownWarnings(console.error);
});
