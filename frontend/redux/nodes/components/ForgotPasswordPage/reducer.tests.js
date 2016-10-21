import configureStore from 'redux-mock-store';
import expect from 'expect';
import thunk from 'redux-thunk';

import {
  clearForgotPasswordErrors,
  forgotPasswordAction,
  forgotPasswordRequestAction,
  forgotPasswordSuccessAction,
  forgotPasswordErrorAction,
} from './actions';
import { invalidForgotPasswordRequest, validForgotPasswordRequest } from '../../../../test/mocks';
import reducer, { initialState } from './reducer';

describe('ForgotPasswordPage - reducer', () => {
  describe('initial state', () => {
    it('sets the initial state', () => {
      expect(reducer(undefined, { type: 'FAKE-ACTION' })).toEqual(initialState);
    });
  });

  describe('clearForgotPasswordErrors', () => {
    it('changes the loading state to true', () => {
      const errorState = {
        ...initialState,
        error: 'Something went wrong',
      };

      expect(reducer(errorState, clearForgotPasswordErrors)).toEqual({
        ...errorState,
        error: null,
      });
    });
  });

  describe('forgotPasswordRequestAction', () => {
    it('changes the loading state to true', () => {
      expect(reducer(initialState, forgotPasswordRequestAction)).toEqual({
        ...initialState,
        loading: true,
      });
    });
  });

  describe('forgotPasswordSuccessAction', () => {
    it('changes the loading state to false and emailSent to true', () => {
      const email = 'hi@thegnar.co';

      expect(reducer(initialState, forgotPasswordSuccessAction(email))).toEqual({
        ...initialState,
        email,
        loading: false,
      });
    });
  });

  describe('forgotPasswordErrorAction', () => {
    it('changes the loading state to false and sets the error state', () => {
      const error = 'There was an error with your request';

      expect(reducer(initialState, forgotPasswordErrorAction(error))).toEqual({
        ...initialState,
        error,
        loading: false,
      });
    });
  });

  describe('forgotPasswordAction', () => {
    const middlewares = [thunk];
    const mockStore = configureStore(middlewares);

    it('dispatches the appropriate actions when successful', (done) => {
      const formData = { email: 'hi@thegnar.co' };
      const request = validForgotPasswordRequest();
      const store = mockStore({});

      store.dispatch(forgotPasswordAction(formData))
        .then(() => {
          const actions = store.getActions();

          expect(actions).toInclude(forgotPasswordRequestAction);
          expect(actions).toInclude(forgotPasswordSuccessAction(formData.email));
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });

    it('dispatches the appropriate actions when unsuccessful', (done) => {
      const formData = { email: 'hi@thegnar.co' };
      const error = 'Something went wrong';
      const invalidRequest = invalidForgotPasswordRequest(error);
      const store = mockStore({});

      store.dispatch(forgotPasswordAction(formData))
        .then(done)
        .catch((errorResponse) => {
          const actions = store.getActions();
          const { response } = errorResponse;

          expect(response).toEqual({ error });
          expect(actions).toInclude(forgotPasswordErrorAction(error));
          expect(invalidRequest.isDone()).toEqual(true);
          done();
        });
    });
  });
});
