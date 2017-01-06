import expect, { restoreSpies, spyOn } from 'expect';

import * as Kolide from 'kolide';

import { reduxMockStore } from 'test/helpers';

import {
  requirePasswordReset,
  REQUIRE_PASSWORD_RESET_REQUEST,
  REQUIRE_PASSWORD_RESET_FAILURE,
  REQUIRE_PASSWORD_RESET_SUCCESS,
} from './actions';

const store = { entities: { invites: {}, users: {} } };
const user = { id: 1, email: 'zwass@kolide.co', force_password_reset: false };

describe('Users - actions', () => {
  describe('dispatching the require password reset action', () => {
    describe('successful request', () => {
      beforeEach(() => {
        spyOn(Kolide.default, 'requirePasswordReset').andCall(() => {
          return Promise.resolve({ ...user, force_password_reset: true });
        });
      });

      afterEach(restoreSpies);

      it('calls the resetFunc', () => {
        const mockStore = reduxMockStore(store);

        return mockStore.dispatch(requirePasswordReset(user, { require: true }))
          .then(() => {
            expect(Kolide.default.requirePasswordReset).toHaveBeenCalledWith(user, { require: true });
          });
      });

      it('dispatches the correct actions', () => {
        const mockStore = reduxMockStore(store);

        const expectedActions = [
          { type: REQUIRE_PASSWORD_RESET_REQUEST },
          {
            type: REQUIRE_PASSWORD_RESET_SUCCESS,
            payload: { user: { ...user, force_password_reset: true } },
          },
        ];

        return mockStore.dispatch(requirePasswordReset(user, { require: true }))
          .then(() => {
            expect(mockStore.getActions()).toEqual(expectedActions);
          });
      });
    });

    describe('unsuccessful request', () => {
      const errors = [
        {
          name: 'base',
          reason: 'Unable to require password reset',
        },
      ];
      const errorResponse = {
        message: {
          message: 'Unable to require password reset',
          errors,
        },
      };

      beforeEach(() => {
        spyOn(Kolide.default, 'requirePasswordReset').andCall(() => {
          return Promise.reject(errorResponse);
        });
      });

      afterEach(restoreSpies);

      it('calls the resetFunc', () => {
        const mockStore = reduxMockStore(store);

        return mockStore.dispatch(requirePasswordReset(user, { require: true }))
          .then(() => {
            throw new Error('promise should have failed');
          })
          .catch(() => {
            expect(Kolide.default.requirePasswordReset).toHaveBeenCalledWith(user, { require: true });
          });
      });

      it('dispatches the correct actions', () => {
        const mockStore = reduxMockStore(store);

        const expectedActions = [
          { type: REQUIRE_PASSWORD_RESET_REQUEST },
          {
            type: REQUIRE_PASSWORD_RESET_FAILURE,
            payload: { errors: { base: 'Unable to require password reset' } },
          },
        ];

        return mockStore.dispatch(requirePasswordReset(user, { require: true }))
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
