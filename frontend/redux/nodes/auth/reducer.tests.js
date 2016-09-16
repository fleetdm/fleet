import configureStore from 'redux-mock-store';
import expect from 'expect';
import thunk from 'redux-thunk';
import authMiddleware from '../../middlewares/auth';
import kolide from '../../../kolide';
import local from '../../../utilities/local';
import { loginRequest, LOGIN_REQUEST, LOGIN_SUCCESS, loginUser } from './actions';
import reducer, { initialState } from './reducer';
import { validLoginRequest, validUser } from '../../../test/mocks';

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
      const loginRequestMock = validLoginRequest();
      store.dispatch(loginUser(formData))
        .then(user => {
          expect(loginRequestMock.isDone()).toEqual(true);
          expect(local.getItem('auth_token')).toEqual(user.token);
          done();
        })
        .catch(done);
    });

    it('returns the authenticated user', (done) => {
      validLoginRequest();

      store.dispatch(loginUser(formData))
        .then(user => {
          expect(user).toEqual(validUser);
          done();
        })
        .catch(done);
    });

    it('sets the users auth token in local storage', (done) => {
      validLoginRequest();

      store.dispatch(loginUser(formData))
        .then(user => {
          expect(local.getItem('auth_token')).toEqual(user.token);
          done();
        })
        .catch(done);
    });

    it('sets the api client bearerToken', (done) => {
      validLoginRequest();

      store.dispatch(loginUser(formData))
        .then(user => {
          expect(kolide.bearerToken).toEqual(user.token);
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
});
