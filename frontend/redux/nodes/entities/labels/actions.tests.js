import expect, { restoreSpies, spyOn } from 'expect';

import * as Kolide from 'kolide';
import labelActions from 'redux/nodes/entities/labels/actions';
import { labelStub } from 'test/stubs';
import { reduxMockStore } from 'test/helpers';

const defaultLabelState = { loading: false, errors: {}, data: {} };
const store = { entities: { labels: defaultLabelState } };

describe('Labels - actions', () => {
  afterEach(restoreSpies);

  describe('#silentLoadAll', () => {
    const { silentLoadAll } = labelActions;

    describe('successful request', () => {
      beforeEach(() => {
        spyOn(Kolide.default.labels, 'loadAll').andReturn(Promise.resolve([labelStub]));
      });


      it('does not call the LOAD_REQUEST action', (done) => {
        const mockStore = reduxMockStore(store);
        const expectedActionTypes = ['labels_LOAD_ALL_SUCCESS'];

        mockStore.dispatch(silentLoadAll())
          .then(() => {
            const actionTypes = mockStore.getActions().map(a => a.type);

            expect(actionTypes).toEqual(expectedActionTypes);
            done();
          })
          .catch(done);
      });
    });

    describe('unsuccessful request', () => {
      beforeEach(() => {
        const errors = {
          message: {
            message: 'Failed validation',
            errors: [{ base: 'Cannot load all labels' }],
          },
        };

        spyOn(Kolide.default.labels, 'loadAll').andReturn(Promise.reject([errors]));
      });


      it('does not call the LOAD_REQUEST action', (done) => {
        const mockStore = reduxMockStore(store);
        const expectedActionTypes = ['labels_LOAD_FAILURE'];

        mockStore.dispatch(silentLoadAll())
          .then(done)
          .catch(() => {
            const actionTypes = mockStore.getActions().map(a => a.type);

            expect(actionTypes).toEqual(expectedActionTypes);
            done();
          });
      });
    });
  });
});
