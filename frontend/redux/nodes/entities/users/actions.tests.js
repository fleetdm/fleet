import Fleet from "fleet";

import { reduxMockStore } from "test/helpers";
import { updateUserSuccess } from "redux/nodes/auth/actions";

import {
  changePassword,
  confirmEmailChange,
  enableUser,
  requirePasswordReset,
  REQUIRE_PASSWORD_RESET_FAILURE,
  REQUIRE_PASSWORD_RESET_SUCCESS,
  updateAdmin,
} from "./actions";
import config from "./config";

const store = { entities: { invites: {}, users: {} } };
const user = { id: 1, email: "zwass@Fleet.co", force_password_reset: false };

describe("Users - actions", () => {
  describe("enableUser", () => {
    describe("successful request", () => {
      beforeEach(() => {
        jest.spyOn(Fleet.users, "enable").mockImplementation(() => {
          return Promise.resolve({ ...user, enabled: true });
        });
      });

      it("calls the API", () => {
        const mockStore = reduxMockStore(store);

        return mockStore
          .dispatch(enableUser(user, { enabled: true }))
          .then(() => {
            expect(Fleet.users.enable).toHaveBeenCalledWith(user, {
              enabled: true,
            });
          });
      });

      it("dispatches the correct actions", (done) => {
        const mockStore = reduxMockStore(store);
        const responseOrder = [user.id];

        mockStore
          .dispatch(enableUser(user, { enabled: true }))
          .then(() => {
            const dispatchedActions = mockStore.getActions();

            expect(dispatchedActions).toEqual([
              config.actions.updateSuccess(
                {
                  users: {
                    [user.id]: { ...user, enabled: true },
                  },
                },
                responseOrder
              ),
            ]);

            done();
          })
          .catch(done);
      });
    });

    describe("unsuccessful request", () => {
      const errors = [
        {
          name: "base",
          reason: "Unable to enable the user",
        },
      ];
      const errorResponse = {
        status: 422,
        message: {
          message: "Unable to enable the user",
          errors,
        },
      };
      beforeEach(() => {
        jest.spyOn(Fleet.users, "enable").mockImplementation(() => {
          return Promise.reject(errorResponse);
        });
      });

      it("calls the API", (done) => {
        const mockStore = reduxMockStore(store);

        mockStore
          .dispatch(enableUser(user, { enabled: true }))
          .then(done)
          .catch(() => {
            expect(Fleet.users.enable).toHaveBeenCalledWith(user, {
              enabled: true,
            });

            done();
          });
      });

      it("dispatches the correct actions", (done) => {
        const mockStore = reduxMockStore(store);

        mockStore
          .dispatch(enableUser(user, { enabled: true }))
          .then(done)
          .catch(() => {
            const dispatchedActions = mockStore.getActions();

            expect(dispatchedActions).toEqual([
              config.actions.updateFailure({
                base: "Unable to enable the user",
                http_status: 422,
              }),
            ]);

            done();
          });
      });
    });
  });

  describe("changePassword", () => {
    const passwordParams = {
      old_password: "p@ssword",
      new_password: "password",
    };
    const changePasswordAction = changePassword(user, passwordParams);

    describe("successful request", () => {
      beforeEach(() => {
        jest.spyOn(Fleet.users, "changePassword").mockImplementation(() => {
          return Promise.resolve({});
        });
      });

      it("calls the API", (done) => {
        const mockStore = reduxMockStore(store);

        mockStore
          .dispatch(changePasswordAction)
          .then(() => {
            expect(Fleet.users.changePassword).toHaveBeenCalledWith(
              passwordParams
            );
            done();
          })
          .catch(done);
      });

      it("dispatches the correct actions", (done) => {
        const mockStore = reduxMockStore(store);
        const responseOrder = [user.id];

        mockStore
          .dispatch(changePasswordAction)
          .then(() => {
            const dispatchedActions = mockStore.getActions();

            expect(dispatchedActions).toEqual([
              config.actions.updateRequest(),
              config.actions.updateSuccess(
                {
                  users: {
                    [user.id]: user,
                  },
                },
                responseOrder
              ),
            ]);

            done();
          })
          .catch(done);
      });
    });

    describe("unsuccessful request", () => {
      const errors = [
        {
          name: "base",
          reason: "Unable to change password",
        },
      ];
      const errorResponse = {
        status: 422,
        message: {
          message: "Unable to change password",
          errors,
        },
      };
      beforeEach(() => {
        jest.spyOn(Fleet.users, "changePassword").mockImplementation(() => {
          return Promise.reject(errorResponse);
        });
      });

      it("calls the API", (done) => {
        const mockStore = reduxMockStore(store);

        mockStore
          .dispatch(changePasswordAction)
          .then(done)
          .catch(() => {
            expect(Fleet.users.changePassword).toHaveBeenCalledWith(
              passwordParams
            );

            done();
          });
      });

      it("dispatches the correct actions", (done) => {
        const mockStore = reduxMockStore(store);

        mockStore
          .dispatch(changePasswordAction)
          .then(done)
          .catch(() => {
            const dispatchedActions = mockStore.getActions();

            expect(dispatchedActions).toEqual([
              config.actions.updateRequest(),
              config.actions.updateFailure({
                base: "Unable to change password",
                http_status: 422,
              }),
            ]);

            done();
          });
      });
    });
  });

  describe("confirmEmailChange", () => {
    const token = "KFBR392";
    const updatedUser = { ...user, email: "new@email.com" };

    describe("successful request", () => {
      beforeEach(() => {
        jest.spyOn(Fleet.users, "confirmEmailChange").mockImplementation(() => {
          return Promise.resolve(updatedUser);
        });
      });

      it("calls the API", (done) => {
        const mockStore = reduxMockStore(store);

        mockStore
          .dispatch(confirmEmailChange(user, token))
          .then(() => {
            expect(Fleet.users.confirmEmailChange).toHaveBeenCalledWith(
              user,
              token
            );
            done();
          })
          .catch(done);
      });

      it("dispatches the correct actions", (done) => {
        const mockStore = reduxMockStore(store);
        const responseOrder = [user.id];

        mockStore
          .dispatch(confirmEmailChange(user, token))
          .then(() => {
            const dispatchedActions = mockStore.getActions();

            expect(dispatchedActions).toEqual([
              config.actions.loadRequest(),
              config.actions.updateSuccess(
                {
                  users: {
                    [user.id]: updatedUser,
                  },
                },
                responseOrder
              ),
              updateUserSuccess(updatedUser),
            ]);

            done();
          })
          .catch(done);
      });
    });

    describe("unsuccessful request", () => {
      const errors = [
        {
          name: "base",
          reason: "Unable to confirm your email address",
        },
      ];
      const errorResponse = {
        status: 422,
        message: {
          message: "Unable to confirm email address",
          errors,
        },
      };
      beforeEach(() => {
        jest.spyOn(Fleet.users, "confirmEmailChange").mockImplementation(() => {
          return Promise.reject(errorResponse);
        });
        jest.spyOn(Fleet.sessions, "destroy").mockImplementation(() => {
          return Promise.resolve({});
        });
      });

      it("calls the API", (done) => {
        const mockStore = reduxMockStore(store);

        mockStore
          .dispatch(confirmEmailChange(user, token))
          .then(() => {
            expect(Fleet.users.confirmEmailChange).toHaveBeenCalledWith(
              user,
              token
            );
            done();
          })
          .catch(done);
      });

      it("dispatches the correct actions", (done) => {
        const mockStore = reduxMockStore(store);

        mockStore
          .dispatch(confirmEmailChange(user, token))
          .then(() => {
            const dispatchedActions = mockStore.getActions();

            expect(dispatchedActions).toEqual([
              config.actions.loadRequest(),
              config.actions.updateFailure({
                base: "Unable to confirm your email address",
                http_status: 422,
              }),
              { type: "LOGOUT_REQUEST" },
              {
                type: "@@router/CALL_HISTORY_METHOD",
                payload: {
                  method: "push",
                  args: ["/login"],
                },
              },
              { type: "LOGOUT_SUCCESS" },
            ]);

            done();
          })
          .catch(done);
      });
    });
  });

  describe("updateAdmin", () => {
    describe("successful request", () => {
      beforeEach(() => {
        jest.spyOn(Fleet.users, "updateAdmin").mockImplementation(() => {
          return Promise.resolve({ ...user, admin: true });
        });
      });

      it("calls the API", (done) => {
        const mockStore = reduxMockStore(store);

        mockStore
          .dispatch(updateAdmin(user, { admin: true }))
          .then(() => {
            expect(Fleet.users.updateAdmin).toHaveBeenCalledWith(user, {
              admin: true,
            });
            done();
          })
          .catch(done);
      });

      it("dispatches the correct actions", (done) => {
        const mockStore = reduxMockStore(store);
        const responseOrder = [user.id];

        mockStore
          .dispatch(updateAdmin(user, { admin: true }))
          .then(() => {
            const dispatchedActions = mockStore.getActions();

            expect(dispatchedActions).toEqual([
              config.actions.updateSuccess(
                {
                  users: {
                    [user.id]: { ...user, admin: true },
                  },
                },
                responseOrder
              ),
            ]);

            done();
          })
          .catch(done);
      });
    });

    describe("unsuccessful request", () => {
      const errors = [
        {
          name: "base",
          reason: "Unable to make the user an admin",
        },
      ];
      const errorResponse = {
        status: 422,
        message: {
          message: "Unable to make the user an admin",
          errors,
        },
      };
      beforeEach(() => {
        jest.spyOn(Fleet.users, "updateAdmin").mockImplementation(() => {
          return Promise.reject(errorResponse);
        });
      });

      it("calls the API", (done) => {
        const mockStore = reduxMockStore(store);

        mockStore
          .dispatch(updateAdmin(user, { admin: true }))
          .then(done)
          .catch(() => {
            expect(Fleet.users.updateAdmin).toHaveBeenCalledWith(user, {
              admin: true,
            });

            done();
          });
      });

      it("dispatches the correct actions", (done) => {
        const mockStore = reduxMockStore(store);

        mockStore
          .dispatch(updateAdmin(user, { admin: true }))
          .then(done)
          .catch(() => {
            const dispatchedActions = mockStore.getActions();

            expect(dispatchedActions).toEqual([
              config.actions.updateFailure({
                base: "Unable to make the user an admin",
                http_status: 422,
              }),
            ]);

            done();
          });
      });
    });
  });

  describe("dispatching the require password reset action", () => {
    describe("successful request", () => {
      beforeEach(() => {
        jest
          .spyOn(Fleet.users, "requirePasswordReset")
          .mockImplementation(() => {
            return Promise.resolve({ ...user, force_password_reset: true });
          });
      });

      it("calls the resetFunc", () => {
        const mockStore = reduxMockStore(store);

        return mockStore
          .dispatch(requirePasswordReset(user, { require: true }))
          .then(() => {
            expect(Fleet.users.requirePasswordReset).toHaveBeenCalledWith(
              user,
              { require: true }
            );
          });
      });

      it("dispatches the correct actions", () => {
        const mockStore = reduxMockStore(store);

        const expectedActions = [
          {
            type: REQUIRE_PASSWORD_RESET_SUCCESS,
            payload: { user: { ...user, force_password_reset: true } },
          },
        ];

        return mockStore
          .dispatch(requirePasswordReset(user, { require: true }))
          .then(() => {
            expect(mockStore.getActions()).toEqual(expectedActions);
          });
      });
    });

    describe("unsuccessful request", () => {
      const errors = [
        {
          name: "base",
          reason: "Unable to require password reset",
        },
      ];
      const errorResponse = {
        status: 422,
        message: {
          message: "Unable to require password reset",
          errors,
        },
      };

      beforeEach(() => {
        jest
          .spyOn(Fleet.users, "requirePasswordReset")
          .mockImplementation(() => {
            return Promise.reject(errorResponse);
          });
      });

      it("calls the resetFunc", () => {
        const mockStore = reduxMockStore(store);

        return mockStore
          .dispatch(requirePasswordReset(user, { require: true }))
          .then(() => {
            throw new Error("promise should have failed");
          })
          .catch(() => {
            expect(Fleet.users.requirePasswordReset).toHaveBeenCalledWith(
              user,
              { require: true }
            );
          });
      });

      it("dispatches the correct actions", () => {
        const mockStore = reduxMockStore(store);

        const expectedActions = [
          {
            type: REQUIRE_PASSWORD_RESET_FAILURE,
            payload: {
              errors: {
                base: "Unable to require password reset",
                http_status: 422,
              },
            },
          },
        ];

        return mockStore
          .dispatch(requirePasswordReset(user, { require: true }))
          .then(() => {
            throw new Error("promise should have failed");
          })
          .catch(() => {
            expect(mockStore.getActions()).toEqual(expectedActions);
          });
      });
    });
  });
});
