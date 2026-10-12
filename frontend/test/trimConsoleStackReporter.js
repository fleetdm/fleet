// Jest captures the full JS call stack for every console.warn/error call in a
// test, most of which is react-dom/react/testing-library internals rather than
// our own code. Reporters run in Jest's main process (unlike setupFilesAfterEnv,
// which runs in each test file's sandboxed module registry), so this is the one
// place that can rewrite that stack before Jest's default reporter prints it,
// while still going through Jest's normal color-coded console output.
function withoutNodeModulesLines(text) {
  return text
    .split("\n")
    .filter((line) => !line.includes("node_modules"))
    .join("\n");
}

class TrimConsoleStackReporter {
  // Jest instantiates reporters with `new`, so this must be an instance method
  // even though it doesn't need `this`.
  // eslint-disable-next-line class-methods-use-this
  onTestResult(_test, testResult) {
    testResult.console?.forEach((entry) => {
      // `origin` is Jest's appended JS call stack; `message` is the warning
      // text itself, which for React dev warnings includes its own component-
      // stack listing (e.g. third-party wrapper components like
      // QueryClientProvider) that can also point into node_modules.
      entry.origin = withoutNodeModulesLines(entry.origin);
      entry.message = withoutNodeModulesLines(entry.message);
    });
  }
}

module.exports = TrimConsoleStackReporter;
