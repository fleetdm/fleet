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
  "uuid",
].join("|");

const config = {
  rootDir: "../../",
  moduleDirectories: ["node_modules", "frontend"],
  testEnvironment: "jest-fixed-jsdom",
  moduleNameMapper: {
    "\\.(sh|ps1)$": "<rootDir>/frontend/__mocks__/fileMock.js",
    "\\.(css|scss|sass)$": "identity-obj-proxy",
    "#minpath": "<rootDir>/node_modules/vfile/lib/minpath.browser.js",
    "#minproc": "<rootDir>/node_modules/vfile/lib/minproc.browser.js",
    "#minurl": "<rootDir>/node_modules/vfile/lib/minurl.browser.js",
    // The editor barrels wrap their component in React.lazy so ace stays out
    // of the entry chunk. jsdom has no chunk to fetch, so that boundary adds no
    // coverage here — only Suspense resolutions landing outside act(), which
    // makes any suite rendering an editor timing-flaky. Point tests straight at
    // the components; the split itself is verified by the build and in-browser.
    "^components/SQLEditor$":
      "<rootDir>/frontend/components/SQLEditor/SQLEditor",
    "^components/Editor$": "<rootDir>/frontend/components/Editor/Editor",
    "^components/YamlAce$": "<rootDir>/frontend/components/YamlAce/YamlAce",
  },
  transform: {
    "^.+\\.[jt]sx?$": "babel-jest",
    "\\.(jpg|jpeg|png|gif|eot|otf|webp|svg|ttf|woff|woff2|mp4|webm|wav|mp3|m4a|aac|oga)$":
      "<rootDir>/frontend/test/fileTransform.js",
  },
  testMatch: ["**/*tests.[jt]s?(x)"],
  // Setting this replaces Jest's default, so /node_modules/ must be restated.
  coveragePathIgnorePatterns: [
    "/node_modules/",
    "<rootDir>/frontend/utilities/osquery_sql_parser/osquery_sql_parser.generated.js",
  ],
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
