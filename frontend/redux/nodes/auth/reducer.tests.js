import configureStore from "redux-mock-store";
import { find } from "lodash";
import thunk from "redux-thunk";

import apiHelpers from "fleet/helpers";
import authMiddleware from "redux/middlewares/auth";
import Fleet from "fleet";
import local from "utilities/local";
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
} from "redux/nodes/auth/actions";
import reducer, { initialState } from "redux/nodes/auth/reducer";
import { userStub } from "test/stubs";
import sessionMocks from "test/mocks/session_mocks";

describe("Auth - reducer", () => {
  it("sets the initial state", () => {
    const state = reducer(undefined, { type: "FOO" });

    expect(state).toEqual(initialState);
  });

  it("changes loading to true for the userLogin action", () => {
    const state = reducer(initialState, loginRequest);

    expect(state).toEqual({
      ...initialState,
      loading: true,
    });
  });

  describe("loginUser action", () => {
    const bearerToken = "expected-bearer-token";
    const formData = {
      email: "username@example.com",
      password: "p@ssw0rd",
    };
    const middlewares = [thunk, authMiddleware];
    const mockStore = configureStore(middlewares);
    const store = mockStore({});

    it("calls the api login endpoint", (done) => {
      const loginRequestMock = sessionMocks.create.valid(bearerToken, formData);

      store
        .dispatch(loginUser(formData))
        .then(() => {
          const loginSuccessAction = find(store.getActions(), {
            type: "LOGIN_SUCCESS",
          });

          expect(loginSuccessAction.payload.token).toEqual(bearerToken);
          expect(loginRequestMock.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });

    it("returns the authenticated user", (done) => {
      sessionMocks.create.valid(bearerToken, formData);

      store
        .dispatch(loginUser(formData))
        .then((user) => {
          expect(user).toEqual(apiHelpers.addGravatarUrlToResource(userStub));
          done();
        })
        .catch(done);
    });

    it("sets the users auth token in local storage", (done) => {
      sessionMocks.create.valid(bearerToken, formData);

      store
        .dispatch(loginUser(formData))
        .then(() => {
          expect(local.getItem("auth_token")).toEqual(bearerToken);
          done();
        })
        .catch(done);
    });

    it("sets the api client bearerToken", (done) => {
      sessionMocks.create.valid(bearerToken, formData);

      store
        .dispatch(loginUser(formData))
        .then(() => {
          expect(Fleet.bearerToken).toEqual(bearerToken);
          done();
        })
        .catch(done);
    });

    it("dispatches LOGIN_REQUEST and LOGIN_SUCCESS actions", (done) => {
      sessionMocks.create.valid(bearerToken, formData);

      store
        .dispatch(loginUser(formData))
        .then(() => {
          const actionTypes = store.getActions().map((a) => a.type);
          expect(actionTypes).toContainEqual(LOGIN_REQUEST, LOGIN_SUCCESS);
          done();
        })
        .catch(done);
    });
  });

  describe("logoutUser action", () => {
    const bearerToken = "ABC123";
    const middlewares = [thunk, authMiddleware];
    const mockStore = configureStore(middlewares);
    const store = mockStore({});

    beforeEach(() => {
      local.setItem("auth_token", bearerToken);
      Fleet.setBearerToken(bearerToken);
    });

    it("calls the api logout endpoint", (done) => {
      const logoutRequestMock = sessionMocks.destroy.valid(bearerToken);

      store
        .dispatch(logoutUser())
        .then(() => {
          expect(logoutRequestMock.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });

    it("removes the users auth token from local storage", (done) => {
      sessionMocks.destroy.valid(bearerToken);

      store
        .dispatch(logoutUser())
        .then(() => {
          expect(local.getItem("auth_token")).toBeFalsy();
          done();
        })
        .catch(done);
    });

    it("clears the api client bearerToken", (done) => {
      sessionMocks.destroy.valid(bearerToken);

      store
        .dispatch(logoutUser())
        .then(() => {
          expect(Fleet.bearerToken).toBeFalsy();
          done();
        })
        .catch(done);
    });

    it("dispatches LOGOUT_REQUEST and LOGOUT_SUCCESS actions", (done) => {
      sessionMocks.destroy.valid(bearerToken);

      store
        .dispatch(logoutUser())
        .then(() => {
          const actionTypes = store.getActions().map((a) => a.type);
          expect(actionTypes).toContainEqual(LOGOUT_REQUEST, LOGOUT_SUCCESS);
          done();
        })
        .catch(done);
    });
  });

  describe("perform required password reset", () => {
    const user = {
      id: 1,
      email: "zwass@Fleet.co",
      force_password_reset: true,
    };

    it("updates state when request is dispatched", () => {
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

    it("updates state when request is successful", () => {
      const initState = {
        ...initialState,
        user,
        loading: true,
      };
      const newUser = { ...user, force_password_reset: false };
      const newState = reducer(
        initState,
        performRequiredPasswordResetSuccess(newUser)
      );

      expect(newState).toEqual({
        ...initState,
        loading: false,
        user: newUser,
      });
    });

    it("updates state when request fails", () => {
      const initState = {
        ...initialState,
        loading: true,
      };
      const errors = { base: "Unable to reset password" };
      const newState = reducer(
        initState,
        performRequiredPasswordResetFailure(errors)
      );

      expect(newState).toEqual({
        ...initState,
        errors,
        loading: false,
      });
    });
  });
});
