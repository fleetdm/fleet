import expect from 'expect';

import {
  clearResetPasswordErrors,
  resetPassword,
  resetPasswordRequest,
  resetPasswordSuccess,
  resetPasswordError,
} from './actions';
import { invalidResetPasswordRequest, validResetPasswordRequest } from '../../../../test/mocks';
import reducer, { initialState } from './reducer';
import { reduxMockStore } from '../../../../test/helpers';

describe('ResetPasswordPage - reducer', () => {
  describe('initial state', () => {
    it('sets the initial state', () => {
      expect(reducer(undefined, { type: 'FAKE-ACTION' })).toEqual(initialState);
    });
  });

  describe('clearResetPasswordErrors', () => {
    it('changes the loading state to true', () => {
      const errorState = {
        ...initialState,
        error: 'Something went wrong',
      };

      expect(reducer(errorState, clearResetPasswordErrors)).toEqual({
        ...errorState,
        error: null,
      });
    });
  });

  describe('resetPasswordRequest', () => {
    it('changes the loading state to true', () => {
      expect(reducer(initialState, resetPasswordRequest)).toEqual({
        ...initialState,
        loading: true,
      });
    });
  });

  describe('resetPasswordSuccess', () => {
    it('changes the loading state to false and errors to null', () => {
      const loadingStateWithError = {
        loading: true,
        error: 'Something went wrong',
      };

      expect(reducer(loadingStateWithError, resetPasswordSuccess)).toEqual({
        loading: false,
        error: null,
      });
    });
  });

  describe('resetPasswordError', () => {
    it('changes the loading state to false and sets the error state', () => {
      const error = 'There was an error with your request';

      expect(reducer(initialState, resetPasswordError(error))).toEqual({
        ...initialState,
        error,
        loading: false,
      });
    });
  });

  describe('resetPassword', () => {
    const newPassword = 'p@ssw0rd';

    it('dispatches the appropriate actions when successful', (done) => {
      const token = 'valid-password-reset-token';
      const formData = {
        new_password: newPassword,
        password_reset_token: token,
      };
      const request = validResetPasswordRequest(newPassword, token);
      const store = reduxMockStore();

      store.dispatch(resetPassword(formData))
        .then(() => {
          const actions = store.getActions();

          expect(actions).toInclude(resetPasswordRequest);
          expect(actions).toInclude(resetPasswordSuccess);
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });

    it('dispatches the appropriate actions when unsuccessful', (done) => {
      const token = 'invalid-password-reset-token';

      const formData = {
        new_password: newPassword,
        password_reset_token: token,
      };
      const error = 'Something went wrong';
      const invalidRequest = invalidResetPasswordRequest(newPassword, token, error);
      const store = reduxMockStore();

      store.dispatch(resetPassword(formData))
        .then(done)
        .catch(errorResponse => {
          const actions = store.getActions();
          const { response } = errorResponse;

          expect(response).toEqual({ error });
          expect(actions).toInclude(resetPasswordError(error));
          expect(invalidRequest.isDone()).toEqual(true);
          done();
        });
    });
  });
});
