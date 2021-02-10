import 'regenerator-runtime/runtime';
import { configure } from 'enzyme';
import Adapter from 'enzyme-adapter-react-16';
import nock from 'nock';

// Uncomment for verbose unhandled promise rejection warnings
// process.on('unhandledRejection', (reason) => {
//   console.error('REJECTION', reason);
// });

nock.disableNetConnect();

// Many tests will output unhandled promise rejection warnings if this is not
// included to mock the common HTTP request.
beforeEach(() => {
  nock('http://localhost:8080')
    .post('/api/v1/fleet/targets', () => true)
    .reply(200, {
      targets_count: 1234,
      targets: [
        {
          id: 3,
          label: 'OS X El Capitan 10.11',
          name: 'osx-10.11',
          platform: 'darwin',
          target_type: 'hosts',
        },
      ],
    });
});

afterEach(nock.cleanAll);

configure({ adapter: new Adapter() });

global.document.queryCommandEnabled = () => { return true; };
global.document.execCommand = () => { return true; };
global.window.getSelection = () => {
  return {
    removeAllRanges: () => { return true; },
  };
};
global.window.URL = new URL('http://localhost:8080');
global.navigator = global.window.navigator;
window.URL.createObjectURL = () => undefined;

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

global.localStorage = mockStorage();
window.localStorage = global.localStorage;


afterEach(nock.cleanAll);
