import expect from 'expect';
import nock from 'nock';
import { find } from 'lodash';

import { reduxMockStore } from 'test/helpers';
import { validGetQueryRequest, invalidGetQueryRequest } from 'test/mocks';
import helpers from 'components/queries/QueryPageWrapper/helpers';

describe('QueryPageWrapper - helpers', () => {
  afterEach(() => { nock.cleanAll(); });

  const queryID = '10';

  describe('#fetchQuery', () => {
    const { fetchQuery } = helpers;
    const bearerToken = 'abc123';

    beforeEach(() => {
      global.localStorage.setItem('KOLIDE::auth_token', bearerToken);
    });

    context('when the API call is successful', () => {
      it('dispatches a load successful action', (done) => {
        validGetQueryRequest(bearerToken, queryID);
        const mockStore = reduxMockStore();

        fetchQuery(mockStore.dispatch, queryID)
          .then(() => {
            const dispatchedActions = mockStore.getActions().map((action) => { return action.type; });
            expect(dispatchedActions).toInclude('queries_LOAD_SUCCESS');

            done();
          });
      });
    });

    context('when the API call is unsuccessful', () => {
      it('pushes to the new query page', (done) => {
        invalidGetQueryRequest(bearerToken, queryID);
        const mockStore = reduxMockStore();

        fetchQuery(mockStore.dispatch, queryID)
          .then(() => {
            const dispatchedActions = mockStore.getActions();
            const locationChangeAction = find(dispatchedActions, { type: '@@router/CALL_HISTORY_METHOD' });
            expect(locationChangeAction).toExist();
            expect(locationChangeAction.payload).toEqual({
              method: 'push',
              args: ['/queries/new'],
            });

            done();
          });
      });

      it('renders a flash error message', (done) => {
        invalidGetQueryRequest(bearerToken, queryID);
        const mockStore = reduxMockStore();

        fetchQuery(mockStore.dispatch, queryID)
          .then(() => {
            const dispatchedActions = mockStore.getActions();
            const flashMessageAction = find(dispatchedActions, { type: 'RENDER_FLASH' });

            expect(flashMessageAction).toExist();
            expect(flashMessageAction.payload).toInclude({
              alertType: 'error',
              message: 'Resource not found',
            });

            done();
          })
          .catch(done);
      });
    });
  });
});
