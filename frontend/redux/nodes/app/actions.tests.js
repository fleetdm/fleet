import expect from 'expect';

import { CONFIG_START, CONFIG_SUCCESS, getConfig } from './actions';
import Kolide from '../../../kolide';
import { reduxMockStore } from '../../../test/helpers';
import { validGetConfigRequest } from '../../../test/mocks';

describe('App - actions', () => {
  describe('getConfig action', () => {
    const store = reduxMockStore({});

    it('calls the api config endpoint', (done) => {
      const bearerToken = 'abc123';
      const request = validGetConfigRequest(bearerToken);

      Kolide.setBearerToken(bearerToken);
      store.dispatch(getConfig())
        .then(() => {
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });

    it('dispatches CONFIG_START & CONFIG_SUCCESS actions', (done) => {
      const bearerToken = 'abc123';
      validGetConfigRequest(bearerToken);

      Kolide.setBearerToken(bearerToken);
      store.dispatch(getConfig())
        .then(() => {
          const actions = store.getActions()
            .map((action) => { return action.type; });

          expect(actions).toInclude(CONFIG_START);
          expect(actions).toInclude(CONFIG_SUCCESS);
          done();
        })
        .catch(done);
    });
  });
});
