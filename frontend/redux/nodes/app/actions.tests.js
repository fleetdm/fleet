import expect from 'expect';

import { CONFIG_START, CONFIG_SUCCESS, getConfig, updateConfig } from 'redux/nodes/app/actions';
import { configStub } from 'test/stubs';
import { frontendFormattedConfig } from 'redux/nodes/app/helpers';
import Kolide from 'kolide';
import { reduxMockStore } from 'test/helpers';
import { validGetConfigRequest, validUpdateConfigRequest } from 'test/mocks';

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

  describe('updateConfig action', () => {
    const store = reduxMockStore({});
    const configFormData = frontendFormattedConfig(configStub);

    it('calls the api update config endpoint', (done) => {
      const bearerToken = 'abc123';
      const request = validUpdateConfigRequest(bearerToken);

      Kolide.setBearerToken(bearerToken);
      store.dispatch(updateConfig(configFormData))
        .then(() => {
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });

    it('dispatches CONFIG_START & CONFIG_SUCCESS actions', (done) => {
      const bearerToken = 'abc123';
      validUpdateConfigRequest(bearerToken);

      Kolide.setBearerToken(bearerToken);
      store.dispatch(updateConfig(configFormData))
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
