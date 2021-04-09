import configureStore from "redux-mock-store";
import thunk from "redux-thunk";

import {
  clearForgotPasswordErrors,
  forgotPasswordAction,
  forgotPasswordRequestAction,
  forgotPasswordSuccessAction,
  forgotPasswordErrorAction,
} from "redux/nodes/components/ForgotPasswordPage/actions";
import userMocks from "test/mocks/user_mocks";
import reducer, {
  initialState,
} from "redux/nodes/components/ForgotPasswordPage/reducer";

describe("ForgotPasswordPage - reducer", () => {
  describe("initial state", () => {
    it("sets the initial state", () => {
      expect(reducer(undefined, { type: "FAKE-ACTION" })).toEqual(initialState);
    });
  });

  describe("clearForgotPasswordErrors", () => {
    it("resets the errors key to an empty object", () => {
      const errorState = {
        ...initialState,
        errors: { base: "Something went wrong" },
      };

      expect(reducer(errorState, clearForgotPasswordErrors)).toEqual({
        ...errorState,
        errors: {},
      });
    });
  });

  describe("forgotPasswordRequestAction", () => {
    it("changes the loading state to true", () => {
      expect(reducer(initialState, forgotPasswordRequestAction)).toEqual({
        ...initialState,
        loading: true,
      });
    });
  });

  describe("forgotPasswordSuccessAction", () => {
    it("changes the loading state to false and emailSent to true", () => {
      const email = "hi@thegnar.co";

      expect(reducer(initialState, forgotPasswordSuccessAction(email))).toEqual(
        {
          ...initialState,
          email,
          loading: false,
        }
      );
    });
  });

  describe("forgotPasswordErrorAction", () => {
    it("changes the loading state to false and sets the error state", () => {
      const errors = { base: "There was an error with your request" };

      expect(reducer(initialState, forgotPasswordErrorAction(errors))).toEqual({
        ...initialState,
        errors,
        loading: false,
      });
    });
  });

  describe("forgotPasswordAction", () => {
    const middlewares = [thunk];
    const mockStore = configureStore(middlewares);

    it("dispatches the appropriate actions when successful", (done) => {
      const formData = { email: "hi@thegnar.co" };
      const request = userMocks.forgotPassword.valid();
      const store = mockStore({});

      store
        .dispatch(forgotPasswordAction(formData))
        .then(() => {
          const actions = store.getActions();

          expect(actions).toContainEqual(forgotPasswordRequestAction);
          expect(actions).toContainEqual(
            forgotPasswordSuccessAction(formData.email)
          );
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });

    it("dispatches the appropriate actions when unsuccessful", (done) => {
      const formData = { email: "hi@thegnar.co" };
      const error = { name: "base", reason: "Something went wrong" };
      const errorResponse = { errors: [error] };
      const invalidRequest = userMocks.forgotPassword.invalid(errorResponse);
      const store = mockStore({});

      store
        .dispatch(forgotPasswordAction(formData))
        .then(done)
        .catch(() => {
          const actions = store.getActions();

          expect(actions).toContainEqual(
            forgotPasswordErrorAction({
              base: "Something went wrong",
              http_status: 422,
            })
          );
          expect(invalidRequest.isDone()).toEqual(true);
          done();
        });
    });
  });
});
