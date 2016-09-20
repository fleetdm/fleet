import jsdom from 'jsdom';

const doc = jsdom.jsdom('<!doctype html><html><body></body></html>', {
  url: 'http://localhost:8080/foo',
});

global.document = doc;
global.window = doc.defaultView;
global.navigator = global.window.navigator;

function mockStorage() {
  let storage = {};

  return {
    setItem(key, value = '') {
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
    clear () {
      storage = {};
    },
  };
}

global.localStorage = window.localStorage = mockStorage();
