import expect, { restoreSpies, spyOn } from 'expect';

import Kolide from 'kolide';

import { reduxMockStore } from 'test/helpers';

import {
  resetOptions,
  resetOptionsStart,
  resetOptionsSuccess,
} from './actions';

const store = { entities: { config_options: {} } };
const options = [
  { id: 1, name: 'option1', type: 'int', value: 10 },
  { id: 2, name: 'option2', type: 'string', value: 'wappa' },
];

describe('Options - actions', () => {
  describe('resetOptions', () => {
    describe('successful request', () => {
      beforeEach(() => {
        spyOn(Kolide.configOptions, 'reset').andCall(() => {
          return Promise.resolve(options);
        });
      });

      afterEach(restoreSpies);

      it('calls the API', () => {
        const mockStore = reduxMockStore(store);
        return mockStore.dispatch(resetOptions())
          .then(() => {
            expect(Kolide.configOptions.reset).toHaveBeenCalled();
          });
      });

      it('dispatches the correct actions', (done) => {
        const mockStore = reduxMockStore(store);
        mockStore.dispatch(resetOptions())
          .then(() => {
            const dispatchedActions = mockStore.getActions();

            expect(dispatchedActions).toEqual([
              resetOptionsStart,
              resetOptionsSuccess(options),
            ]);

            done();
          })
          .catch(done);
      });
    });
  });
});
