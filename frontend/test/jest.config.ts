import type { Config } from "jest";

const config: Config = {
  rootDir: "../../",
  moduleDirectories: ["node_modules", "frontend"],
  testEnvironment: "jsdom",
  moduleNameMapper: {
    "\\.(jpg|jpeg|png|gif|eot|otf|webp|svg|ttf|woff|woff2|mp4|webm|wav|mp3|m4a|aac|oga)$":
      "<rootDir>/frontend/__mocks__/fileMock.js",
    "\\.(css|scss|sass)$": "identity-obj-proxy",
    "react-markdown":
      "<rootDir>/node_modules/react-markdown/react-markdown.min.js",
  },
  testMatch: ["**/*tests.[jt]s?(x)"],
  setupFilesAfterEnv: ["<rootDir>/frontend/test/test-setup.ts"],
  clearMocks: true,
  testEnvironmentOptions: {
    url: "http://localhost:8080",
  },
};

export default config;
