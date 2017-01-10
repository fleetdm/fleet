import expect, { restoreSpies, spyOn } from 'expect';

import * as Kolide from 'kolide';

import { reduxMockStore } from 'test/helpers';

import {
  performRequiredPasswordReset,
  PERFORM_REQUIRED_PASSWORD_RESET_REQUEST,
  PERFORM_REQUIRED_PASSWORD_RESET_FAILURE,
  PERFORM_REQUIRED_PASSWORD_RESET_SUCCESS,
} from './actions';

const store = { entities: { invites: {}, users: {} } };
const user = { id: 1, email: 'zwass@kolide.co', force_password_reset: false };

describe('Auth - actions', () => {
  describe('dispatching the perform required password reset action', () => {
    describe('successful request', () => {
      beforeEach(() => {
        spyOn(Kolide.default, 'performRequiredPasswordReset').andCall(() => {
          return Promise.resolve({ ...user, force_password_reset: false });
        });
      });

      afterEach(restoreSpies);

      const resetParams = { password: 'foobar' };

      it('calls the resetFunc', () => {
        const mockStore = reduxMockStore(store);

        return mockStore.dispatch(performRequiredPasswordReset(resetParams))
          .then(() => {
            expect(Kolide.default.performRequiredPasswordReset).toHaveBeenCalledWith(resetParams);
          });
      });

      it('dispatches the correct actions', () => {
        const mockStore = reduxMockStore(store);

        const expectedActions = [
          { type: PERFORM_REQUIRED_PASSWORD_RESET_REQUEST },
          {
            type: PERFORM_REQUIRED_PASSWORD_RESET_SUCCESS,
            payload: { user: { ...user, force_password_reset: false } },
          },
        ];

        return mockStore.dispatch(performRequiredPasswordReset(resetParams))
          .then(() => {
            expect(mockStore.getActions()).toEqual(expectedActions);
          });
      });
    });

    describe('unsuccessful request', () => {
      const errors = [
        {
          name: 'base',
          reason: 'Unable to reset password',
        },
      ];
      const errorResponse = {
        message: {
          message: 'Unable to perform reset',
          errors,
        },
      };
      const resetParams = { password: 'foobar' };

      beforeEach(() => {
        spyOn(Kolide.default, 'performRequiredPasswordReset').andCall(() => {
          return Promise.reject(errorResponse);
        });
      });

      afterEach(restoreSpies);

      it('calls the resetFunc', () => {
        const mockStore = reduxMockStore(store);

        return mockStore.dispatch(performRequiredPasswordReset(resetParams))
          .then(() => {
            throw new Error('promise should have failed');
          })
          .catch(() => {
            expect(Kolide.default.performRequiredPasswordReset).toHaveBeenCalledWith(resetParams);
          });
      });

      it('dispatches the correct actions', () => {
        const mockStore = reduxMockStore(store);

        const expectedActions = [
          { type: PERFORM_REQUIRED_PASSWORD_RESET_REQUEST },
          {
            type: PERFORM_REQUIRED_PASSWORD_RESET_FAILURE,
            payload: { errors: { base: 'Unable to reset password' } },
          },
        ];

        return mockStore.dispatch(performRequiredPasswordReset(resetParams))
          .then(() => {
            throw new Error('promise should have failed');
          })
          .catch(() => {
            expect(mockStore.getActions()).toEqual(expectedActions);
          });
      });
    });
  });
});
