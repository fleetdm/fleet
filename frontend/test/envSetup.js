// used for babel polyfills.
import "core-js/stable";
import "regenerator-runtime/runtime";

import { configure } from "enzyme";
import Adapter from "enzyme-adapter-react-16";
import nock from "nock";

// for testing-library utils
import "@testing-library/jest-dom";

// Uncomment for verbose unhandled promise rejection warnings
// process.on('unhandledRejection', (reason) => {
//   console.error('REJECTION', reason);
// });

nock.disableNetConnect();

nock.emitter.on("no match", (req) => {
  console.log("NOCK NO MATCH ", req);
});

// nock.emitter.on('request', (req, interceptor) => {
//   console.error('interceptor matched request: ', req, interceptor)
// });
// nock.emitter.on('replied', (req, interceptor) => {
//   console.error('response replied with nocked payload', req, interceptor)
// });

// Many tests will output unhandled promise rejection warnings if this is not
// included to mock the common HTTP request.

nock("http://localhost:8080")
  .persist()
  .post("/api/v1/fleet/targets")
  .reply(200, {
    targets_count: 1234,
    targets: [
      {
        id: 3,
        label: "OS X El Capitan 10.11",
        name: "osx-10.11",
        platform: "darwin",
        target_type: "hosts",
      },
    ],
  });

nock("http://localhost:8080")
  .persist()
  .get("/api/v1/fleet/status/live_query")
  .reply(200, {});

nock("http://localhost:8080")
  .persist()
  .get("/api/v1/fleet/version")
  .reply(200, {
    version: "3.10.0",
    branch: "master",
    revision: "83d608962af583375bc20c644c5ac4b00b408461",
    go_version: "go1.16.2",
    build_date: "2021-03-31T20:05:51Z",
    build_user: "zwass",
  });

configure({ adapter: new Adapter() });

global.document.queryCommandEnabled = jest.fn();
global.document.execCommand = jest.fn();
global.window.getSelection = () => {
  return {
    removeAllRanges: () => {
      return true;
    },
  };
};
global.window.scrollTo = jest.fn();
global.window.URL = new URL("http://localhost:8080");
global.navigator = global.window.navigator;
window.URL.createObjectURL = () => undefined;

function mockStorage() {
  let storage = {};

  return {
    setItem(key, value = "") {
      storage[key] = value;
    },
    getItem(key) {
      return storage[key];
    },
    removeItem(key) {
      delete storage[key];
    },
    get length() {
      return Object.keys(storage).length;
    },
    key(i) {
      return Object.keys(storage)[i] || null;
    },
    clear() {
      storage = {};
    },
  };
}

global.localStorage = mockStorage();
window.localStorage = global.localStorage;
