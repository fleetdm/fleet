/*
 * For a detailed explanation regarding each configuration property and type check, visit:
 * https://jestjs.io/docs/configuration
 */

export default {
  // Automatically clear mock calls, instances, contexts and results before every test
  clearMocks: true,
  // Indicates whether the coverage information should be collected while executing the test
  collectCoverage: true,
  // The directory where Jest should output its coverage files
  coverageDirectory: "coverage",
  // Indicates which provider should be used to instrument code for coverage
  coverageProvider: "v8",
  extensionsToTreatAsEsm: [".ts", ".tsx"],
  // A preset that is used as a base for Jest's configuration
  preset: "ts-jest/presets/default-esm",
  moduleNameMapper: {
    "^(\\.{1,2}/.*)\\.js$": "$1",
  },
  transform: {
    "^.+\\.m?[tj]sx?$": [
      "ts-jest",
      {
        useESM: true,
      },
    ],
  },
  // The paths to modules that run some code to configure or set up the testing environment before
  // each test
  setupFiles: [],
  setupFilesAfterEnv: ["./jest.setup.ts"],
  // The test environment that will be used for testing
  testEnvironment: "./jsdomwithfetch.ts",
  // Define additional global variables
  globals: {
    // Neither jest nor jsdom include the chrome global, so we need to define it here.
    chrome: {
      runtime: {},
      privacy: { network: {}, services: {}, websites: {} },
      idle: {},
      system: { storage: {} },
    },
  },
};
