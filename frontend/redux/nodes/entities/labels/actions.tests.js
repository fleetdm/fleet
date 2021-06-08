import nock from "nock";

import labelActions from "redux/nodes/entities/labels/actions";
import { labelStub } from "test/stubs";
import { reduxMockStore } from "test/helpers";

const defaultLabelState = { loading: false, errors: {}, data: {} };
const store = { entities: { labels: defaultLabelState } };

describe("Labels - actions", () => {
  afterEach(() => {
    nock.cleanAll();
  });

  describe("#silentLoadAll", () => {
    const { silentLoadAll } = labelActions;

    describe("successful request", () => {
      it("does not call the LOAD_REQUEST action", (done) => {
        nock("http://localhost:8080")
          .get("/api/v1/fleet/labels")
          .reply(200, { labels: [labelStub] });

        const mockStore = reduxMockStore(store);
        const expectedActionTypes = ["labels_LOAD_ALL_SUCCESS"];

        mockStore
          .dispatch(silentLoadAll())
          .then(() => {
            const actionTypes = mockStore.getActions().map((a) => a.type);

            expect(actionTypes).toEqual(expectedActionTypes);
            done();
          })
          .catch(() => {
            const actionTypes = mockStore.getActions().map((a) => a.type);

            expect(actionTypes).toEqual(expectedActionTypes);
            done();
          });
      });
    });

    describe("unsuccessful request", () => {
      it("does not call the LOAD_REQUEST action", (done) => {
        const mockStore = reduxMockStore(store);
        const expectedActionTypes = ["labels_LOAD_FAILURE"];
        const errors = {
          message: {
            message: "Failed validation",
            errors: [{ base: "Cannot load all labels" }],
          },
        };

        nock("http://localhost:8080")
          .get("/api/v1/fleet/labels")
          .reply(422, { errors });

        mockStore
          .dispatch(silentLoadAll())
          .then(done)
          .catch(() => {
            const actionTypes = mockStore.getActions().map((a) => a.type);

            expect(actionTypes).toEqual(expectedActionTypes);
            done();
          });
      });
    });
  });
});
