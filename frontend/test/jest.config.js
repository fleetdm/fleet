/* global TransformStream */
/** @type {import('jest').Config} */

const esModules = [
  "react-markdown",
  "vfile",
  "vfile-message",
  "micromark.*",
  "unist-.+",
  "unified",
  "bail",
  "is-plain-obj",
  "trough",
  "remark-.+",
  "mdast-util-.+",
  "parse-entities",
  "character-entities",
  "property-information",
  "comma-separated-tokens",
  "hast-util-whitespace",
  "remark-.+",
  "space-separated-tokens",
  "decode-named-character-reference",
  "ccount",
  "escape-string-regexp",
  "markdown-table",
  "trim-lines",
  "hast-util-.+",
  "html-url-attributes",
  "devlop",
  "estree-.+",
  "estree-util-.+",
  "periscopic",
  "is-reference",
  "stringify-entities",
  "character-entities-html4",
  "character-entities-legacy",
  "zwitch",
  "longest-streak",
].join("|");

const config = {
  rootDir: "../../",
  moduleDirectories: ["node_modules", "frontend"],
  testEnvironment: "jest-fixed-jsdom",
  moduleNameMapper: {
    "\\.(jpg|jpeg|png|gif|eot|otf|webp|svg|ttf|woff|woff2|mp4|webm|wav|mp3|m4a|aac|oga)$":
      "<rootDir>/frontend/__mocks__/fileMock.js",
    "\\.(sh|ps1)$": "<rootDir>/frontend/__mocks__/fileMock.js",
    "\\.(css|scss|sass)$": "identity-obj-proxy",
<<<<<<< HEAD
    "#minpath": "<rootDir>/node_modules/vfile/lib/minpath.browser.js",
    "#minproc": "<rootDir>/node_modules/vfile/lib/minproc.browser.js",
    "#minurl": "<rootDir>/node_modules/vfile/lib/minurl.browser.js",
=======
    "^node-sql-parser$":
      "<rootDir>/node_modules/@sgress454/node-sql-parser/umd/sqlite.umd.js",
>>>>>>> main
  },
  testMatch: ["**/*tests.[jt]s?(x)"],
  setupFilesAfterEnv: ["<rootDir>/frontend/test/test-setup.ts"],
  clearMocks: true,
  testEnvironmentOptions: {
    url: "http://fleettest.test:9876",
    customExportConditions: [""],
  },
  transformIgnorePatterns: [`/node_modules/(?!(${esModules})/)`],
  globals: {
    TransformStream,
    featureFlags: {},
  },
};

module.exports = config;
