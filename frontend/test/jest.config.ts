import type { Config } from "jest";

const esModules = [
  "react-markdown",
  "vfile",
  "vfile-message",
  "micromark.+",
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
].join("|");

const config: Config = {
  rootDir: "../../",
  moduleDirectories: ["node_modules", "frontend"],
  testEnvironment: "jsdom",
  moduleNameMapper: {
    "\\.(jpg|jpeg|png|gif|eot|otf|webp|svg|ttf|woff|woff2|mp4|webm|wav|mp3|m4a|aac|oga)$":
      "<rootDir>/frontend/__mocks__/fileMock.js",
    "\\.(css|scss|sass)$": "identity-obj-proxy",
    // "react-markdown":
    //   "<rootDir>/node_modules/react-markdown/react-markdown.min.js",
    // "remark-gfm": "<rootDir>/node_modules/remark-gfm/index.js",
    // "micromark-extension-gfm":
    //   "<rootDir>/node_models/micromark-extension-gfm/index.js",
  },
  testMatch: ["**/*tests.[jt]s?(x)"],
  setupFilesAfterEnv: ["<rootDir>/frontend/test/test-setup.ts"],
  clearMocks: true,
  testEnvironmentOptions: {
    url: "http://localhost:8080",
  },
  // transformIgnorePatterns: ["node_modules/(?!react-markdown/)"],
  transformIgnorePatterns: [`/node_modules/(?!(${esModules})/)`],
};

export default config;
