import expect, { spyOn, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import nock from 'nock';

import { connectedComponent, reduxMockStore } from 'test/helpers';
import helpers from 'components/queries/QueryPageWrapper/helpers';
import QueryPageWrapper from 'components/queries/QueryPageWrapper';
import { validGetQueryRequest } from 'test/mocks';

const bearerToken = 'abc123';
const storeWithoutQuery = {
  entities: {
    queries: {
      data: {},
    },
  },
};

describe('QueryPageWrapper - component', () => {
  beforeEach(() => {
    global.localStorage.setItem('KOLIDE::auth_token', bearerToken);
  });

  afterEach(() => {
    restoreSpies();
    nock.cleanAll();
  });

  describe('/queries/:id', () => {
    const queryID = '10';
    const locationProp = { params: { id: queryID } };

    it('dispatches an action to get the query when there is no query', () => {
      validGetQueryRequest(bearerToken, queryID);

      const mockStore = reduxMockStore(storeWithoutQuery);

      mount(connectedComponent(QueryPageWrapper, { mockStore, props: locationProp }));

      const dispatchedActions = mockStore.getActions().map((action) => { return action.type; });
      expect(dispatchedActions).toInclude('queries_LOAD_REQUEST');
    });

    it('calls the fetchQuery helper function', () => {
      validGetQueryRequest(bearerToken, queryID);

      const fetchQuerySpy = spyOn(helpers, 'fetchQuery');
      const mockStore = reduxMockStore(storeWithoutQuery);

      mount(connectedComponent(QueryPageWrapper, { mockStore, props: locationProp }));

      expect(fetchQuerySpy).toHaveBeenCalled();

      restoreSpies();
    });
  });
});
