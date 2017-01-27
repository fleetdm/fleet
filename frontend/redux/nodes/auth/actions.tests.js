import expect, { restoreSpies, spyOn } from 'expect';

import * as Kolide from 'kolide';
import userActions from 'redux/nodes/entities/users/actions';

import { reduxMockStore } from 'test/helpers';
import { userStub } from 'test/stubs';

import {
  performRequiredPasswordReset,
  PERFORM_REQUIRED_PASSWORD_RESET_REQUEST,
  PERFORM_REQUIRED_PASSWORD_RESET_FAILURE,
  PERFORM_REQUIRED_PASSWORD_RESET_SUCCESS,
  updateUser,
} from './actions';

const store = { entities: { invites: {}, users: {} } };
const user = { ...userStub, id: 1, email: 'zwass@kolide.co', force_password_reset: false };

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

  describe('#updateUser', () => {
    it('calls the user update action', () => {
      const updatedAttrs = { name: 'Jerry Garcia' };
      const updatedUser = { ...userStub, ...updatedAttrs };
      const mockStore = reduxMockStore(store);
      const expectedActions = [
        { type: 'UPDATE_USER_REQUEST' },
        { type: 'UPDATE_USER_SUCCESS', payload: { user: updatedUser } },
      ];

      spyOn(userActions, 'update').andReturn(() => Promise.resolve(updatedUser));

      return mockStore.dispatch(updateUser(userStub, updatedAttrs))
        .then(() => {
          expect(mockStore.getActions()).toEqual(expectedActions);
        })
        .catch(() => {
          throw new Error(`Expected ${mockStore.getActions()} to equal ${expectedActions}`);
        });
    });
  });
});
