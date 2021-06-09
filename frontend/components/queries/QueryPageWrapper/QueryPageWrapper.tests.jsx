import { mount } from "enzyme";
import nock from "nock";

import { connectedComponent, reduxMockStore } from "test/helpers";
import helpers from "components/queries/QueryPageWrapper/helpers";
import QueryPageWrapper from "components/queries/QueryPageWrapper";
import mocks from "test/mocks";

const { queries: queryMocks } = mocks;
const bearerToken = "abc123";
const storeWithoutQuery = {
  entities: {
    queries: {
      data: {},
    },
  },
};

describe("QueryPageWrapper - component", () => {
  beforeEach(() => {
    global.localStorage.setItem("FLEET::auth_token", bearerToken);
  });

  afterEach(() => {
    nock.cleanAll();
  });

  describe("/queries/:id", () => {
    const queryID = "10";
    const locationProp = { params: { id: queryID } };

    it("dispatches an action to get the query when there is no query", () => {
      queryMocks.load.valid(bearerToken, queryID);

      const mockStore = reduxMockStore(storeWithoutQuery);

      mount(
        connectedComponent(QueryPageWrapper, { mockStore, props: locationProp })
      );

      const dispatchedActions = mockStore.getActions().map((action) => {
        return action.type;
      });
      expect(dispatchedActions).toContainEqual("queries_LOAD_REQUEST");
    });

    it("calls the fetchQuery helper function", () => {
      queryMocks.load.valid(bearerToken, queryID);

      const fetchQuerySpy = jest.spyOn(helpers, "fetchQuery");
      const mockStore = reduxMockStore(storeWithoutQuery);

      mount(
        connectedComponent(QueryPageWrapper, { mockStore, props: locationProp })
      );

      expect(fetchQuerySpy).toHaveBeenCalled();
    });
  });
});
