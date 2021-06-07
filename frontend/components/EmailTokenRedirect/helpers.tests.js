import { reduxMockStore } from "test/helpers";

import helpers from "components/EmailTokenRedirect/helpers";
import Fleet from "fleet";
import { userStub } from "test/stubs";

describe("EmailTokenRedirect - helpers", () => {
  describe("#confirmEmailChage", () => {
    const { confirmEmailChange } = helpers;
    const token = "KFBR392";
    const authStore = {
      auth: {
        user: userStub,
      },
    };

    describe("successfully dispatching the confirmEmailChange action", () => {
      beforeEach(() => {
        jest
          .spyOn(Fleet.users, "confirmEmailChange")
          .mockImplementation(() =>
            Promise.resolve({ ...userStub, email: "new@email.com" })
          );
      });

      it("pushes the user to the settings page", () => {
        const mockStore = reduxMockStore(authStore);
        const { dispatch } = mockStore;

        return confirmEmailChange(dispatch, userStub, token).then(() => {
          const dispatchedActions = mockStore.getActions();

          expect(dispatchedActions).toContainEqual({
            type: "@@router/CALL_HISTORY_METHOD",
            payload: {
              method: "push",
              args: ["/profile"],
            },
          });
        });
      });
    });

    describe("unsuccessfully dispatching the confirmEmailChange action", () => {
      beforeEach(() => {
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

        jest
          .spyOn(Fleet.users, "confirmEmailChange")
          .mockImplementation(() => Promise.reject(errorResponse));
      });

      it("pushes the user to the login page", () => {
        const mockStore = reduxMockStore(authStore);
        const { dispatch } = mockStore;

        return confirmEmailChange(dispatch, userStub, token)
          .then()
          .catch(() => {
            const dispatchedActions = mockStore.getActions();

            expect(dispatchedActions).toContainEqual({
              type: "@@router/CALL_HISTORY_METHOD",
              payload: {
                method: "push",
                args: ["/login"],
              },
            });
          });
      });
    });

    describe("when the user or token are not present", () => {
      it("does not dispatch any actions when the user is not present", (done) => {
        const mockStore = reduxMockStore(authStore);
        const { dispatch } = mockStore;

        confirmEmailChange(dispatch, undefined, token)
          .then(() => {
            const dispatchedActions = mockStore.getActions();

            expect(dispatchedActions).toEqual([]);

            done();
          })
          .catch(done);
      });

      it("does not dispatch any actions when the token is not present", (done) => {
        const mockStore = reduxMockStore(authStore);
        const { dispatch } = mockStore;

        confirmEmailChange(dispatch, userStub, undefined)
          .then(() => {
            const dispatchedActions = mockStore.getActions();

            expect(dispatchedActions).toEqual([]);

            done();
          })
          .catch(done);
      });
    });
  });
});
