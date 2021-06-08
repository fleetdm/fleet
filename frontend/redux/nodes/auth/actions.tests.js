import Fleet from "fleet";
import userActions from "redux/nodes/entities/users/actions";

import { reduxMockStore } from "test/helpers";
import { userStub } from "test/stubs";

import {
  performRequiredPasswordReset,
  PERFORM_REQUIRED_PASSWORD_RESET_REQUEST,
  PERFORM_REQUIRED_PASSWORD_RESET_FAILURE,
  PERFORM_REQUIRED_PASSWORD_RESET_SUCCESS,
  SSO_REDIRECT_REQUEST,
  SSO_REDIRECT_SUCCESS,
  updateUser,
  ssoRedirect,
} from "./actions";

const store = { entities: { invites: {}, users: {} } };
const user = {
  ...userStub,
  id: 1,
  email: "zwass@Fleet.co",
  force_password_reset: false,
};

describe("Auth - actions", () => {
  describe("#ssoRedirect", () => {
    const ssoURL = "http://salesforce.idp.com";
    const relayURL = "/";

    describe("successful request", () => {
      beforeEach(() => {
        jest
          .spyOn(Fleet.sessions, "initializeSSO")
          .mockImplementation(() => Promise.resolve({ url: ssoURL }));
      });

      it("calls the API", () => {
        const mockStore = reduxMockStore(store);

        mockStore
          .dispatch(ssoRedirect(relayURL))
          .then(() => {
            expect(Fleet.sessions.initializeSSO).toHaveBeenCalledWith(relayURL);
          })
          .catch(() => {
            expect(Fleet.sessions.initializeSSO).toHaveBeenCalledWith(relayURL);
          });
      });

      it("executes to correct actions", () => {
        const mockStore = reduxMockStore(store);
        const actions = [
          { type: SSO_REDIRECT_REQUEST },
          { type: SSO_REDIRECT_SUCCESS, payload: { ssoRedirectURL: ssoURL } },
        ];

        return mockStore
          .dispatch(ssoRedirect(relayURL))
          .then(() => {
            expect(mockStore.getActions()).toEqual(actions);
          })
          .catch(() => {
            expect(mockStore.getActions()).toEqual(actions);
          });
      });

      it("retrieves redirect url", () => {
        const mockStore = reduxMockStore(store);
        return mockStore
          .dispatch(ssoRedirect(relayURL))
          .then((result) => {
            expect(ssoURL).toEqual(result.payload.ssoRedirectURL);
          })
          .catch((result) => {
            expect(ssoURL).toEqual(result);
          });
      });
    });
  });

  describe("dispatching the perform required password reset action", () => {
    describe("successful request", () => {
      beforeEach(() => {
        jest
          .spyOn(Fleet.users, "performRequiredPasswordReset")
          .mockImplementation(() => {
            return Promise.resolve({ ...user, force_password_reset: false });
          });
      });

      const resetParams = { password: "foobar" };

      it("calls the resetFunc", () => {
        const mockStore = reduxMockStore(store);

        return mockStore
          .dispatch(performRequiredPasswordReset(resetParams))
          .then(() => {
            expect(
              Fleet.users.performRequiredPasswordReset
            ).toHaveBeenCalledWith(resetParams);
          });
      });

      it("dispatches the correct actions", () => {
        const mockStore = reduxMockStore(store);

        const expectedActions = [
          { type: PERFORM_REQUIRED_PASSWORD_RESET_REQUEST },
          {
            type: PERFORM_REQUIRED_PASSWORD_RESET_SUCCESS,
            payload: { user: { ...user, force_password_reset: false } },
          },
        ];

        return mockStore
          .dispatch(performRequiredPasswordReset(resetParams))
          .then(() => {
            expect(mockStore.getActions()).toEqual(expectedActions);
          });
      });
    });

    describe("unsuccessful request", () => {
      const errors = [
        {
          name: "base",
          reason: "Unable to reset password",
        },
      ];
      const errorResponse = {
        status: 422,
        message: {
          message: "Unable to perform reset",
          errors,
        },
      };
      const resetParams = { password: "foobar" };

      beforeEach(() => {
        jest
          .spyOn(Fleet.users, "performRequiredPasswordReset")
          .mockImplementation(() => {
            return Promise.reject(errorResponse);
          });
      });

      it("calls the resetFunc", () => {
        const mockStore = reduxMockStore(store);

        return mockStore
          .dispatch(performRequiredPasswordReset(resetParams))
          .then(() => {
            throw new Error("promise should have failed");
          })
          .catch(() => {
            expect(
              Fleet.users.performRequiredPasswordReset
            ).toHaveBeenCalledWith(resetParams);
          });
      });

      it("dispatches the correct actions", () => {
        const mockStore = reduxMockStore(store);

        const expectedActions = [
          { type: PERFORM_REQUIRED_PASSWORD_RESET_REQUEST },
          {
            type: PERFORM_REQUIRED_PASSWORD_RESET_FAILURE,
            payload: {
              errors: { base: "Unable to reset password", http_status: 422 },
            },
          },
        ];

        return mockStore
          .dispatch(performRequiredPasswordReset(resetParams))
          .then(() => {
            throw new Error("promise should have failed");
          })
          .catch(() => {
            expect(mockStore.getActions()).toEqual(expectedActions);
          });
      });
    });
  });

  describe("#updateUser", () => {
    it("calls the user update action", () => {
      const updatedAttrs = { name: "Jerry Garcia" };
      const updatedUser = { ...userStub, ...updatedAttrs };
      const mockStore = reduxMockStore(store);
      const expectedActions = [
        { type: "UPDATE_USER_SUCCESS", payload: { user: updatedUser } },
      ];

      jest
        .spyOn(userActions, "silentUpdate")
        .mockImplementation(() => () => Promise.resolve(updatedUser));

      return mockStore
        .dispatch(updateUser(userStub, updatedAttrs))
        .then(() => {
          expect(mockStore.getActions()).toEqual(expectedActions);
        })
        .catch(() => {
          throw new Error(
            `Expected ${mockStore.getActions()} to equal ${expectedActions}`
          );
        });
    });
  });
});
