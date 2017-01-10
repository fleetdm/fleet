import configureStore from 'redux-mock-store';
import expect from 'expect';
import { find } from 'lodash';
import thunk from 'redux-thunk';

import authMiddleware from '../../middlewares/auth';
import kolide from '../../../kolide';
import local from '../../../utilities/local';
import {
  loginRequest,
  LOGIN_REQUEST,
  LOGIN_SUCCESS,
  loginUser,
  logoutUser,
  LOGOUT_REQUEST,
  LOGOUT_SUCCESS,
    performRequiredPasswordResetRequest,
  performRequiredPasswordResetSuccess,
  performRequiredPasswordResetFailure,
} from './actions';
import reducer, { initialState } from './reducer';
import {
  validLoginRequest,
  validLogoutRequest,
  validUser,
} from '../../../test/mocks';

describe('Auth - reducer', () => {
  it('sets the initial state', () => {
    const state = reducer(undefined, { type: 'FOO' });

    expect(state).toEqual(initialState);
  });

  it('changes loading to true for the userLogin action', () => {
    const state = reducer(initialState, loginRequest);

    expect(state).toEqual({
      ...initialState,
      loading: true,
    });
  });

  context('loginUser action', () => {
    const formData = {
      username: 'username',
      password: 'p@ssw0rd',
    };
    const middlewares = [thunk, authMiddleware];
    const mockStore = configureStore(middlewares);
    const store = mockStore({});


    it('calls the api login endpoint', (done) => {
      const expectedBearerToken = 'expected-bearer-token';
      const loginRequestMock = validLoginRequest(expectedBearerToken);

      store.dispatch(loginUser(formData))
        .then(() => {
          const loginSuccessAction = find(store.getActions(), { type: 'LOGIN_SUCCESS' });

          expect(loginSuccessAction.payload.token).toEqual(expectedBearerToken);
          expect(loginRequestMock.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });

    it('returns the authenticated user', (done) => {
      validLoginRequest();

      store.dispatch(loginUser(formData))
        .then((user) => {
          expect(user).toEqual(validUser);
          done();
        })
        .catch(done);
    });

    it('sets the users auth token in local storage', (done) => {
      const expectedBearerToken = 'expected-bearer-token';
      validLoginRequest(expectedBearerToken);

      store.dispatch(loginUser(formData))
        .then(() => {
          expect(local.getItem('auth_token')).toEqual(expectedBearerToken);
          done();
        })
        .catch(done);
    });

    it('sets the api client bearerToken', (done) => {
      const expectedBearerToken = 'expected-bearer-token';
      validLoginRequest(expectedBearerToken);

      store.dispatch(loginUser(formData))
        .then(() => {
          expect(kolide.bearerToken).toEqual(expectedBearerToken);
          done();
        })
        .catch(done);
    });

    it('dispatches LOGIN_REQUEST and LOGIN_SUCCESS actions', (done) => {
      validLoginRequest();

      store.dispatch(loginUser(formData))
        .then(() => {
          const actionTypes = store.getActions().map(a => a.type);
          expect(actionTypes).toInclude(LOGIN_REQUEST, LOGIN_SUCCESS);
          done();
        })
        .catch(done);
    });
  });

  context('logoutUser action', () => {
    const bearerToken = 'ABC123';
    const middlewares = [thunk, authMiddleware];
    const mockStore = configureStore(middlewares);
    const store = mockStore({});

    beforeEach(() => {
      local.setItem('auth_token', bearerToken);
      kolide.setBearerToken(bearerToken);
    });


    it('calls the api logout endpoint', (done) => {
      const logoutRequestMock = validLogoutRequest(bearerToken);
      store.dispatch(logoutUser())
        .then(() => {
          expect(logoutRequestMock.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });

    it('removes the users auth token from local storage', (done) => {
      validLogoutRequest(bearerToken);

      store.dispatch(logoutUser())
        .then(() => {
          expect(local.getItem('auth_token')).toNotExist();
          done();
        })
        .catch(done);
    });

    it('clears the api client bearerToken', (done) => {
      validLogoutRequest(bearerToken);

      store.dispatch(logoutUser())
        .then(() => {
          expect(kolide.bearerToken).toNotExist();
          done();
        })
        .catch(done);
    });

    it('dispatches LOGOUT_REQUEST and LOGOUT_SUCCESS actions', (done) => {
      validLogoutRequest(bearerToken);

      store.dispatch(logoutUser())
        .then(() => {
          const actionTypes = store.getActions().map(a => a.type);
          expect(actionTypes).toInclude(LOGOUT_REQUEST, LOGOUT_SUCCESS);
          done();
        })
        .catch(done);
    });
  });

  context('perform required password reset', () => {
    const user = { id: 1, email: 'zwass@kolide.co', force_password_reset: true };

    it('updates state when request is dispatched', () => {
      const initState = {
        ...initialState,
        user,
      };
      const newState = reducer(initState, performRequiredPasswordResetRequest);

      expect(newState).toEqual({
        ...initState,
        loading: true,
      });
    });

    it('updates state when request is successful', () => {
      const initState = {
        ...initialState,
        user,
        loading: true,
      };
      const newUser = { ...user, force_password_reset: false };
      const newState = reducer(initState, performRequiredPasswordResetSuccess(newUser));

      expect(newState).toEqual({
        ...initState,
        loading: false,
        user: newUser,
      });
    });

    it('updates state when request fails', () => {
      const initState = {
        ...initialState,
        loading: true,
      };
      const errors = { base: 'Unable to reset password' };
      const newState = reducer(initState, performRequiredPasswordResetFailure(errors));

      expect(newState).toEqual({
        ...initState,
        errors,
        loading: false,
      });
    });
  });
});
