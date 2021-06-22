import nock from "nock";
import { find } from "lodash";

import { reduxMockStore } from "test/helpers";
import mocks from "test/mocks";
import helpers from "components/queries/QueryPageWrapper/helpers";

const { queries: queryMocks } = mocks;

describe("QueryPageWrapper - helpers", () => {
  afterEach(() => {
    nock.cleanAll();
  });

  const queryID = "10";

  describe("#fetchQuery", () => {
    const { fetchQuery } = helpers;
    const bearerToken = "abc123";

    beforeEach(() => {
      global.localStorage.setItem("FLEET::auth_token", bearerToken);
    });

    describe("when the API call is successful", () => {
      it("dispatches a load successful action", (done) => {
        queryMocks.load.valid(bearerToken, queryID);
        const mockStore = reduxMockStore();

        fetchQuery(mockStore.dispatch, queryID).then(() => {
          const dispatchedActions = mockStore.getActions().map((action) => {
            return action.type;
          });
          expect(dispatchedActions).toContainEqual("queries_LOAD_SUCCESS");

          done();
        });
      });
    });

    describe("when the API call is unsuccessful", () => {
      it("pushes to the manage queries page", (done) => {
        queryMocks.load.invalid(bearerToken, queryID);
        const mockStore = reduxMockStore();

        fetchQuery(mockStore.dispatch, queryID).then(() => {
          const dispatchedActions = mockStore.getActions();
          const locationChangeAction = find(dispatchedActions, {
            type: "@@router/CALL_HISTORY_METHOD",
          });
          expect(locationChangeAction).toBeTruthy();
          expect(locationChangeAction.payload).toEqual({
            method: "push",
            args: ["/queries/manage"],
          });

          done();
        });
      });

      it("renders a flash error message", (done) => {
        queryMocks.load.invalid(bearerToken, queryID);
        const mockStore = reduxMockStore();

        fetchQuery(mockStore.dispatch, queryID)
          .then(() => {
            const dispatchedActions = mockStore.getActions();
            const flashMessageAction = find(dispatchedActions, {
              type: "RENDER_FLASH",
            });

            expect(flashMessageAction).toBeTruthy();

            if (
              flashMessageAction.payload.message.includes(
                "no rows in result set"
              )
            ) {
              expect(flashMessageAction.payload).toMatchObject({
                alertType: "error",
                message: "The query you requested does not exist in Fleet.",
              });
            } else {
              expect(flashMessageAction.payload).toMatchObject({
                alertType: "error",
                message: "Resource not found",
              });
            }

            done();
          })
          .catch(done);
      });
    });
  });
});
